# Deploy

IaaS, PaaS, system and container orchestration deployment configurations and templates (docker-compose, kubernetes/helm, mesos, terraform, bosh). The AWS deployment uses the [AWS Cloud Development Kit](https://github.com/awslabs/aws-cdk).

- [Docker](#deploy-docker)
- [AWS - ECS](#deploy-aws-ecs)

<a name="deploy-docker"></a>
## Docker

Enter the `docker/` directory for performing these commands.

### Usage

Note: Ensure your configuration is correctly setup to use S3 for storage or you could lose data:

    docker pull tokenized/smartcontractd
    docker run --env-file ./smartcontractd.conf tokenized/smartcontractd

### Building

Build:

    docker build -t tokenized/smartcontractd -f ./Dockerfile ../../../

Run as a local test:

    docker run --rm -it --env-file ./smartcontractd.conf tokenized/smartcontractd

Push to dockerhub:

    docker login

    docker push tokenized/smartcontractd

<a name="deploy-aws-ecs"></a>
## AWS - ECS

Enter the `aws-ecs/` directory for performing these commands.

Refer to the [AWS EC2 deployment](#deploy-aws-ec2) for more detailed instructions, the process is mostly the same.

### Useful CDK Commands

 * `npm run build`   compile typescript to js
 * `npm run watch`   watch for changes and compile
 * `cdk deploy`      deploy this stack to your default AWS account/region
 * `cdk diff`        compare deployed stack with current state
 * `cdk synth`       emits the synthesized CloudFormation template
