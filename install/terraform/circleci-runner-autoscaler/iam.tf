data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "circleci-runners-autoscaler-pd" {
  statement {
    sid    = "AllowAllAutoscaling"
    effect = "Allow"

    resources = [
      "*"
    ]

    actions = [
        "autoscaling:*",
    ]
  }
}

resource "aws_iam_policy" "circleci-runners-autoscaler-policy" {
  name   = "CircleCIRunnersAutoScalerPolicy"
  policy = data.aws_iam_policy_document.circleci-runners-autoscaler-pd.json
}

data "aws_iam_policy_document" "circleci-runners-assume-role-policy" {
  dynamic "statement" {
    for_each = toset(var.oidc_issuers)

    content {
      actions = ["sts:AssumeRoleWithWebIdentity"]
      effect = "Allow"
      
      condition {
        test     = "StringEquals"
        variable = "${statement.value}:aud"

        values = [
          "sts.amazonaws.com"
        ]
      }

      condition {
        test     = "StringEquals"
        variable = "${statement.value}:sub"

        values = [
          "system:serviceaccount:circleci-runners-autoscaler:circleci-runners-autoscaler"
        ]
      }

      principals {
        type        = "Federated"
        identifiers = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/${statement.value}"]
      }
    }
  }
}

resource "aws_iam_role" "circleci-runners-autoscaler-role" {
  name = "CircleCIRunnersAutoScalerRole"

  assume_role_policy = data.aws_iam_policy_document.circleci-runners-assume-role-policy.json
}

resource "aws_iam_role_policy_attachment" "attach-circleci-runners-autoscaler-policy" {
  role       = aws_iam_role.circleci-runners-autoscaler-role.name
  policy_arn = aws_iam_policy.circleci-runners-autoscaler-policy.arn
}
