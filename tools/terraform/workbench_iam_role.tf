# Assume role for the instance
resource "aws_iam_role" "workbench_iam_role" {
    name = "${var.cluster}-workbench-iam-role"
    assume_role_policy = templatefile("${path.module}/iam_policy_assume_role.json", {})
}

# Associate letsencrypt modification policy to the role
resource "aws_iam_role_policy_attachment" "workbench_letsencrypt_route53_policies_attachment" {
    role       = aws_iam_role.workbench_iam_role.name
    policy_arn = aws_iam_policy.letsencrypt_route53_iam_policy.arn
}

# Add the role to the instance profile
resource "aws_iam_instance_profile" "workbench_instance_profile" {
  name  = "workbench_instance_profile"
  role = "${var.cluster}-workbench-iam-role"
}
