# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

terraform {
  required_version = "~> 1.3.0"
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 4.38.0"
    }
  }
}

provider "aws" {
  region = local.region_name
  default_tags {
    tags = merge(local.custom_tags, {
      Arvados = local.cluster_name
      Terraform = true
    })
  }
}

resource "aws_iam_instance_profile" "keepstore_instance_profile" {
  name = "${local.cluster_name}-keepstore-00-iam-role"
  role = data.terraform_remote_state.data-storage.outputs.keepstore_iam_role_name
}

resource "aws_iam_instance_profile" "compute_node_instance_profile" {
  name = "${local.cluster_name}-compute-node-00-iam-role"
  role = local.compute_node_iam_role_name
}

resource "aws_iam_instance_profile" "dispatcher_instance_profile" {
  name = "${local.cluster_name}_dispatcher_instance_profile"
  role = aws_iam_role.cloud_dispatcher_iam_role.name
}

resource "aws_secretsmanager_secret" "ssl_password_secret" {
  name = local.ssl_password_secret_name
  recovery_window_in_days = 0
}

resource "aws_iam_instance_profile" "default_instance_profile" {
  name = "${local.cluster_name}_default_instance_profile"
  role = aws_iam_role.default_iam_role.name
}

resource "aws_instance" "arvados_service" {
  for_each = toset(concat(local.public_hosts, local.private_hosts))
  ami = local.instance_ami_id
  instance_type = try(var.instance_type[each.value], var.instance_type.default)
  user_data = templatefile("user_data.sh", {
    "hostname": each.value,
    "deploy_user": var.deploy_user,
    "ssh_pubkey": file(local.pubkey_path)
  })
  private_ip = local.private_ip[each.value]
  subnet_id = contains(local.user_facing_hosts, each.value) ? local.public_subnet_id : local.private_subnet_id
  vpc_security_group_ids = [ local.arvados_sg_id ]
  iam_instance_profile = try(local.instance_profile[each.value], local.instance_profile.default).name
  tags = {
    Name = "${local.cluster_name}_arvados_service_${each.value}"
  }
  root_block_device {
    volume_type = "gp3"
    volume_size = try(var.instance_volume_size[each.value], var.instance_volume_size.default)
  }
  metadata_options {
    # Sets IMDSv2 to required. Default is "optional".
    http_tokens = "required"
  }
  lifecycle {
    ignore_changes = [
      # Avoids recreating the instance when the latest AMI changes.
      # Use 'terraform taint' or 'terraform apply -replace' to force
      # an AMI change.
      ami,
    ]
  }
}

resource "aws_iam_policy" "compute_node_ebs_autoscaler" {
  name = "${local.cluster_name}_compute_node_ebs_autoscaler"
  policy = jsonencode({
    Version: "2012-10-17",
    Id: "compute-node EBS Autoscaler policy",
    Statement: [{
      Effect: "Allow",
      Action: [
          "ec2:AttachVolume",
          "ec2:DescribeVolumeStatus",
          "ec2:DescribeVolumes",
          "ec2:DescribeTags",
          "ec2:ModifyInstanceAttribute",
          "ec2:DescribeVolumeAttribute",
          "ec2:CreateVolume",
          "ec2:DeleteVolume",
          "ec2:CreateTags"
      ],
      Resource: "*"
    }]
  })
}

resource "aws_iam_policy_attachment" "compute_node_ebs_autoscaler_attachment" {
  name = "${local.cluster_name}_compute_node_ebs_autoscaler_attachment"
  roles = [ local.compute_node_iam_role_name ]
  policy_arn = aws_iam_policy.compute_node_ebs_autoscaler.arn
}

resource "aws_iam_policy" "cloud_dispatcher_ec2_access" {
  name = "${local.cluster_name}_cloud_dispatcher_ec2_access"
  policy = jsonencode({
    Version: "2012-10-17",
    Id: "arvados-dispatch-cloud policy",
    Statement: [{
      Effect: "Allow",
      Action: [
        "ec2:DescribeKeyPairs",
        "ec2:ImportKeyPair",
        "ec2:RunInstances",
        "ec2:DescribeInstances",
        "ec2:CreateTags",
        "ec2:TerminateInstances"
      ],
      Resource: "*"
    },
    {
      Effect: "Allow",
      Action: [
        "iam:PassRole",
      ],
      Resource: "arn:aws:iam::*:role/${aws_iam_instance_profile.compute_node_instance_profile.name}"
    }]
  })
}

resource "aws_iam_role" "cloud_dispatcher_iam_role" {
  name = "${local.cluster_name}-dispatcher-00-iam-role"
  assume_role_policy = "${file("../assumerolepolicy.json")}"
}

resource "aws_iam_policy_attachment" "cloud_dispatcher_ec2_access_attachment" {
  name = "${local.cluster_name}_cloud_dispatcher_ec2_access_attachment"
  roles = [ aws_iam_role.cloud_dispatcher_iam_role.name ]
  policy_arn = aws_iam_policy.cloud_dispatcher_ec2_access.arn
}

resource "aws_eip_association" "eip_assoc" {
  for_each = local.private_only ? [] : toset(local.public_hosts)
  instance_id = aws_instance.arvados_service[each.value].id
  allocation_id = local.eip_id[each.value]
}

resource "aws_iam_role" "default_iam_role" {
  name = "${local.cluster_name}-default-iam-role"
  assume_role_policy = "${file("../assumerolepolicy.json")}"
}

resource "aws_iam_policy" "ssl_privkey_password_access" {
  name = "${local.cluster_name}_ssl_privkey_password_access"
  policy = jsonencode({
    Version: "2012-10-17",
    Statement: [{
      Effect: "Allow",
      Action: "secretsmanager:GetSecretValue",
      Resource: "${aws_secretsmanager_secret.ssl_password_secret.arn}"
    }]
  })
}

# Every service node needs access to the SSL privkey password secret for
# nginx to be able to use it.
resource "aws_iam_policy_attachment" "ssl_privkey_password_access_attachment" {
  name = "${local.cluster_name}_ssl_privkey_password_access_attachment"
  roles = [
    aws_iam_role.cloud_dispatcher_iam_role.name,
    aws_iam_role.default_iam_role.name,
    local.keepstore_iam_role_name,
  ]
  policy_arn = aws_iam_policy.ssl_privkey_password_access.arn
}
