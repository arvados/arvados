# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

data "terraform_remote_state" "vpc" {
  backend = "local"
  config = {
    path = "../vpc/terraform.tfstate"
  }
}

data "terraform_remote_state" "data-storage" {
  backend = "local"
  config = {
    path = "../data-storage/terraform.tfstate"
  }
}

# https://wiki.debian.org/Cloud/AmazonEC2Image/Bookworm
data "aws_ami" "debian-12" {
  most_recent = true
  owners = ["136693071363"]
  filter {
    name   = "name"
    values = ["debian-12-amd64-*"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

data "aws_vpc" "arvados_vpc" {
  id = data.terraform_remote_state.vpc.outputs.arvados_vpc_id
}
