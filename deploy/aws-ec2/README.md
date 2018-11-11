# Deploy - AWS - EC2

Ensure you have an [AWS account](https://portal.aws.amazon.com/billing/signup#/start), you have the [AWS CLI installed](https://docs.aws.amazon.com/cli/latest/userguide/installing.html), and that your AWS CLI [credentials are correctly configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html):

```
aws configure
```

Install [AWS CDK](https://github.com/awslabs/aws-cdk)

```
npm i -g aws-cdk
```

Create or import an SSH Key Pair to allow access to the running server:

* https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html
* https://docs.aws.amazon.com/cli/latest/reference/ec2/create-key-pair.html
* https://docs.aws.amazon.com/cli/latest/reference/ec2/import-key-pair.html
* https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-key-pairs.html

```
aws ec2 import-key-pair --key-name tokenized-ssh --public-key-material file://~/.ssh/tokenized-ssh.pub
```

Make any required configuration changes to `./bin/tokenized.ts` (eg. EC2 instance type/size, key name, etc). In particular, make sure you have updated the `TokenizedStack` properties to define appropriate values in `appConfig`:

```typescript
new TokenizedEC2Stack(app, 'TokenizedEC2Stack', {
    appConfig: {
        OPERATOR_NAME: "Standalone",
        VERSION: "0.1",
        FEE_ADDRESS: "yourfeeaddress",
        FEE_VALUE: 2000,
        NODE_ADDRESS: "1.2.3.4:8333",
        NODE_USER_AGENT: "",
        RPC_HOST: "1.2.3.4:8332",
        RPC_USERNAME: "youruser",
        RPC_PASSWORD: "yoursecretpassword",
        PRIV_KEY: "yourwif",
    },
    ec2InstanceClass: ec2.InstanceClass.T3,
    ec2InstanceSize: ec2.InstanceSize.Micro,
    enableSSH: false,
});
```

Don't forget to compile any changes you made:

```
npm run-script build
# Or in another terminal, watch and recompile with: npm run-script watch
```

Make sure you have already built the `smatcontractd` binary:

```
cd ../../ && make clean prepare deps dist-smartcontractd
# Make sure to return to the deploy directory afterwards: cd ./deploy/aws-ec2
```

Check the differences between the currently deployed stack (if it exists), and the stack as defined in `./bin/tokenized.ts`: 

```
cdk diff
```

If you're happy with the proposed changes, deploy the stack:

```
cdk deploy
```

## Config

`config.template` uses [mustache](https://mustache.github.io/) template syntax to [generate the config file at deploy time](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-init.html#aws-resource-init-files).

## Service

`smartcontract.service.template` is configured for [systemd](https://www.freedesktop.org/software/systemd/man/systemd.service.html) (as [supported on Amazon Linux 2](https://aws.amazon.com/amazon-linux-2/release-notes/#systemd)), and [managed during deploy time](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-init.html#aws-resource-init-services).

## Useful CDK Commands

 * `npm run build`   compile typescript to js
 * `npm run watch`   watch for changes and compile
 * `cdk deploy`      deploy this stack to your default AWS account/region
 * `cdk diff`        compare deployed stack with current state
 * `cdk synth`       emits the synthesized CloudFormation template
