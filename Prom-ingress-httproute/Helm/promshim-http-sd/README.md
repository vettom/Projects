# promshim-http-sd Helm chart
Chart to deploy promshim-http-sd app from ECR. 
Ptometheus HTTP service Discovery app that exeposes list of all ingresses configured in cluster.

## Steps to update and push to ECR
```bash
helm package promshim-http-sd
aws ecr get-login-password  --profile cor-audio --region eu-west-2 | helm registry login --username AWS --password-stdin  701336946071.dkr.ecr.eu-west-2.amazonaws.com
helm push promshim-http-sd-2.0.0.tgz oci://701336946071.dkr.ecr.eu-west-2.amazonaws.com

```

## Helm pull from OCI repo
```bash
helm pull oci://701336946071.dkr.ecr.eu-west-2.amazonaws.com/promshim-http-sd --version <version>
```
