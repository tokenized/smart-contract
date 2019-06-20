#!/usr/bin/env node

import path = require('path');

import cdk = require('@aws-cdk/cdk');
import assets = require('@aws-cdk/assets');
import autoscaling = require('@aws-cdk/aws-autoscaling');
import ec2 = require('@aws-cdk/aws-ec2');
import s3 = require('@aws-cdk/aws-s3');

class TokenizedEC2Stack extends cdk.Stack {
    constructor(parent: cdk.App, name: string, props: TokenizedProps) {
        super(parent, name, props);

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ec2.html
        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-autoscaling.html

        const vpc = new ec2.VpcNetwork(this, 'Tokenized-VPC', {
            natGateways: 0, // Set to 0 because we manually setup our own NAT instance, which is a lot cheaper
            subnetConfiguration: [
                {
                    cidrMask: 26,
                    name: 'Public',
                    subnetType: ec2.SubnetType.Public,
                },
                {
                    name: 'Application',
                    subnetType: ec2.SubnetType.Private,
                },
            ],
            defaultInstanceTenancy: ec2.DefaultInstanceTenancy.Default,
        });

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ec2.html#@aws-cdk/aws-ec2.SecurityGroup
        const natSecurityGroup = new ec2.SecurityGroup(this, 'NATSecurityGroup', {
            vpc,
            groupName: "NATSecurityGroup",
            description: "NAT Instance Security Group",
            allowAllOutbound: true,
        });

        natSecurityGroup.tags.setTag("Name", natSecurityGroup.path);

        // TODO: Is this ok?
        natSecurityGroup.connections.allowFromAnyIPv4(new ec2.TcpAllPorts());

        // Ref: https://docs.aws.amazon.com/vpc/latest/userguide/vpc-network-acls.html#nacl-ephemeral-ports
        // natSecurityGroup.connections.allowFromAnyIPv4(new ec2.TcpPortRange(32768, 65535));

        // if (props.enableSSH) {
        //     // Allow ssh access
        //     // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ec2.html#allowing-connections
        //     natSecurityGroup.connections.allowFromAnyIPv4(new ec2.TcpPort(22));
        // }

        // Ref: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-instance.html
        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ec2.html#@aws-cdk/aws-ec2.SecurityGroupProps
        const natInstance = new ec2.cloudformation.InstanceResource(this, 'NATInstance', {
            imageId: "ami-062c04ec46aecd204", // ap-southeast-2: amzn-ami-vpc-nat-hvm-2018.03.0.20181116-x86_64-ebs
            instanceType: new ec2.InstanceTypePair(props.ec2NATInstanceClass, props.ec2NATInstanceSize).toString(),
            subnetId: vpc.publicSubnets[0].subnetId,
            securityGroupIds: [natSecurityGroup.securityGroupId],
            sourceDestCheck: false, // Required for NAT
            keyName: "tokenized-ssh",
        });

        natInstance.propertyOverrides.tags = [
            {key: "Name", value: `${natInstance.path}`}
        ];

        // Route private subnets through the NAT instance
        vpc.privateSubnets.forEach(subnet => {
            const defaultRoute = subnet.findChild('DefaultRoute') as ec2.cloudformation.RouteResource;
            defaultRoute.propertyOverrides.instanceId = natInstance.instanceId;
        });
        // Ref: https://aws.amazon.com/amazon-linux-2/
        // Ref: https://aws.amazon.com/amazon-linux-2/release-notes/
        const amazonLinux2 = new ec2.AmazonLinuxImage({
            generation: ec2.AmazonLinuxGeneration.AmazonLinux2,
        });

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-autoscaling.html
        const appAsg = new autoscaling.AutoScalingGroup(this, 'AppAutoScalingGroup', {
            vpc,
            minSize: 1,
            maxSize: 1,
            desiredCapacity: 1,
            instanceType: new ec2.InstanceTypePair(props.ec2InstanceClass, props.ec2InstanceSize),
            machineImage: amazonLinux2,
            keyName: "tokenized-ssh",
            vpcPlacement: {
                subnetName: 'Application'
            },
            updateType: autoscaling.UpdateType.ReplacingUpdate,
            // updateType: autoscaling.UpdateType.RollingUpdate,
            // rollingUpdateConfiguration: {
            //     waitOnResourceSignals: true,
            //     pauseTimeSec: 120,
            // }
        });

        // Ref: https://docs.aws.amazon.com/vpc/latest/userguide/vpc-network-acls.html#nacl-ephemeral-ports
        appAsg.connections.allowFromAnyIPv4(new ec2.TcpPortRange(32768, 65535));

        if (props.enableSSH) {
            // Allow ssh access
            // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ec2.html#allowing-connections
            appAsg.connections.allowFromAnyIPv4(new ec2.TcpPort(22));
        }

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_assets.html

        const projectRoot = path.join(__dirname, '../../../');

        const appAsset = new assets.ZipDirectoryAsset(this, 'AppAsset', {
            path: path.join(projectRoot, 'dist')
        });

        const configTemplateAsset = new assets.FileAsset(this, 'ConfigTemplateAsset', {
            path: path.join(__dirname, '../config.template')
        });

        const serviceTemplateAsset = new assets.FileAsset(this, 'ServiceTemplateAsset', {
            path: path.join(__dirname, '../smartcontract.service.template')
        });

        // Allow the server to access uploaded assets
        appAsset.grantRead(appAsg.role);
        configTemplateAsset.grantRead(appAsg.role);
        serviceTemplateAsset.grantRead(appAsg.role);

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-s3.html

        // Setup Node Storage
        let nodeStorageBucket;
        if (typeof props.appConfig.NODE_STORAGE_BUCKET == "undefined") {
            nodeStorageBucket = new s3.Bucket(this, "NodeStorageBucket")
        } else if (props.appConfig.NODE_STORAGE_BUCKET.toLowerCase() == "standalone") {
            this.addError("NODE_STORAGE_BUCKET=standalone means that any data will get lost when the EC2 server is terminated. This is probably not what you want to happen..");
        } else {
            this.addWarning(`The specified NODE_STORAGE_BUCKET (${ props.appConfig.NODE_STORAGE_BUCKET}) is defined externally to this stack. Make sure appropriate permissions are set so we can write to it, or weird things might happen.`);

            nodeStorageBucket = s3.Bucket.import(this, "NodeStorageBucket", {
                bucketName: props.appConfig.NODE_STORAGE_BUCKET
            });
        }

        if (nodeStorageBucket) {
            nodeStorageBucket.grantReadWrite(appAsg.role);
        }

        // Setup Contract Storage
        let contractStorageBucket;
        if (typeof props.appConfig.CONTRACT_STORAGE_BUCKET == "undefined") {
            contractStorageBucket = new s3.Bucket(this, "ContractStorageBucket")
        } else if (props.appConfig.CONTRACT_STORAGE_BUCKET.toLowerCase() == "standalone") {
            this.addError("CONTRACT_STORAGE_BUCKET=standalone means that any data will get lost when the EC2 server is terminated. This is probably not what you want to happen..");
        }
        else {
            this.addWarning(`The specified CONTRACT_STORAGE_BUCKET (${ props.appConfig.CONTRACT_STORAGE_BUCKET}) is defined externally to this stack. Make sure appropriate permissions are set so we can write to it, or weird things might happen.`);

            contractStorageBucket = s3.Bucket.import(this, "ContractStorageBucket", {
                bucketName: props.appConfig.CONTRACT_STORAGE_BUCKET
            });
        }

        if (contractStorageBucket) {
            contractStorageBucket.grantReadWrite(appAsg.role);
        }

        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-init.html
        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-authentication.html
        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/example-templates-autoscaling.html
        // Ref: https://aws.amazon.com/blogs/devops/authenticated-file-downloads-with-cloudformation/
        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-hup.html
        // TODO: Improve this when the following is resolved: https://github.com/awslabs/aws-cdk/issues/777
        const asgLaunchConfig = appAsg.findChild('LaunchConfig') as autoscaling.cloudformation.LaunchConfigurationResource;

        asgLaunchConfig.addDependency(natInstance);
        asgLaunchConfig.addDependency(appAsg.role);

        const cfnhupConfigFilePath = "/etc/cfn/cfn-hup.conf";
        const cfnhupAutoReloadFilePath = "/etc/cfn/hooks.d/cfn-auto-reloader.conf";

        const configFilePath = "/etc/tokenized/env";

        const serviceName = "smartcontractd";
        const serviceFilePath = `/usr/lib/systemd/system/${serviceName}.service`;

        const binPath = "/usr/local/bin";
        const execPath = path.join(binPath, serviceName);

        const stackRegionResource = new cdk.FnJoin("", [
            "--stack=", new cdk.AwsStackName, " ",
            "--region=", new cdk.AwsRegion, " ",
            "--resource=", asgLaunchConfig.logicalId,
        ]);

        const cfnInitCmd = new cdk.FnJoin(" ", [
            "/opt/aws/bin/cfn-init -v", stackRegionResource
        ]);

        let configFileContext = props.appConfig;

        let useAuthBuckets = [
            appAsset.s3BucketName,
            configTemplateAsset.s3BucketName,
            serviceTemplateAsset.s3BucketName,
        ];

        if (nodeStorageBucket) {
            configFileContext.NODE_STORAGE_BUCKET = nodeStorageBucket.bucketName;
            configFileContext.NODE_STORAGE_REGION = new cdk.AwsRegion().resolve();

            useAuthBuckets.push(nodeStorageBucket.bucketName);
        }

        if (contractStorageBucket) {
            configFileContext.CONTRACT_STORAGE_BUCKET = contractStorageBucket.bucketName;
            configFileContext.CONTRACT_STORAGE_REGION = new cdk.AwsRegion().resolve();

            useAuthBuckets.push(contractStorageBucket.bucketName);
        }

        // Note: asset.s3Url didn't work (403), see: https://forums.aws.amazon.com/thread.jspa?threadID=262326#833949
        function assetS3URL(asset: assets.Asset) {
            return `https://${asset.s3BucketName}.s3-${new cdk.AwsRegion}.amazonaws.com/${asset.s3ObjectKey}`
        }

        asgLaunchConfig.addOverride("Metadata", {
            "AWS::CloudFormation::Init": {
                // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-init.html#aws-resource-init-configsets
                "configSets": {
                    "install": [ "main" ],
                    "update": [ "ensureservicestopped" , "main" ]
                },
                "ensureservicestopped": {
                    // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-init.html#aws-resource-init-commands
                    "commands": {
                        "stopservice": {
                            "command": `sudo service ${serviceName} stop`,
                            "ignoreErrors": true,
                        },
                    },
                },
                "main": {
                    // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-init.html#aws-resource-init-sources
                    "sources": {
                        [binPath]: assetS3URL(appAsset),
                    },
                    // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-init.html#aws-resource-init-files
                    "files": {
                        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-hup.html#cfn-hup-config-file
                        [cfnhupConfigFilePath]: {
                            "content": [
                                "[main]",
                                `stack=${new cdk.AwsStackName}`,
                                `region=${new cdk.AwsRegion}`,
                                "interval=5",
                                // "verbose=true",
                            ].join("\n"),
                            "mode"  : "000400",
                            "owner" : "root",
                            "group" : "root"
                        },
                        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-hup.html#cfn-hup-hook-file
                        [cfnhupAutoReloadFilePath]: {
                            "content": [
                                "[cfn-auto-reloader-hook]",
                                "triggers=post.update",
                                `path=Resources.${asgLaunchConfig.logicalId}.Metadata.AWS::CloudFormation::Init`,
                                `action=${cfnInitCmd} --configsets update`,
                                "runas=root",
                            ].join("\n"),
                            "mode"  : "000400",
                            "owner" : "root",
                            "group" : "root"
                        },
                        [configFilePath]: {
                            "source": assetS3URL(configTemplateAsset),
                            "context": configFileContext,
                        },
                        [serviceFilePath]: {
                            "source": assetS3URL(serviceTemplateAsset),
                            "context": {
                                "configFilePath": configFilePath,
                                "execPath": execPath
                            }
                        }
                    },
                    // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-init.html#aws-resource-init-services
                    "services" : {
                        "sysvinit": {
                            "cfn-hup": {
                                "enabled": "true",
                                "ensureRunning": "true",
                                "files": [
                                    cfnhupConfigFilePath,
                                    cfnhupAutoReloadFilePath,
                                ],
                            },
                            [serviceName]: {
                                "enabled": "true",
                                "ensureRunning": "true",
                                "files": [
                                    configFilePath,
                                    serviceFilePath,
                                    execPath,
                                ],
                            },
                        }
                    }
                }
            },
            "AWS::CloudFormation::Authentication": {
                "S3AccessCreds": {
                    "type": "S3",
                    "roleName": appAsg.role.roleName,
                    "buckets": useAuthBuckets,
                }
            }
        });

        // Ref: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html
        // Ref: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/amazon-linux-ami-basics.html#extras-library
        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-helper-scripts-reference.html
        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-init.html
        // Ref: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-signal.html
        appAsg.addUserData(
            "yum install -y aws-cfn-bootstrap;",
            new cdk.FnJoin(" ", [cfnInitCmd, "--configsets install;"]).toString(),
            new cdk.FnJoin(" ", ["/opt/aws/bin/cfn-signal", stackRegionResource, "--exit-code $?;"]).toString()
        );
    }
}

/**
 * Configuration for smartcontractd
 *
 * You should only have to set the NODE_STORAGE_* and CONTRACT_STORAGE_*
 * variables in exceptional circumstances. When left blank, this stack
 * will automatically provision and configure the required resources.
 */
interface AppConfig {
    OPERATOR_NAME: string
    VERSION: string

    START_HASH: string

    RPC_HOST: string
    RPC_USERNAME: string
    RPC_PASSWORD: string

    PRIV_KEY: string
    FEE_ADDRESS: string
    BITCOIN_CHAIN: string

    NODE_ADDRESS: string
    NODE_USER_AGENT: string
    NODE_STORAGE_ROOT?: string
    NODE_STORAGE_BUCKET?: string
    NODE_STORAGE_REGION?: string
    NODE_STORAGE_ACCESS_KEY?: string
    NODE_STORAGE_SECRET?: string

    CONTRACT_STORAGE_ROOT?: string
    CONTRACT_STORAGE_BUCKET?: string
    CONTRACT_STORAGE_REGION?: string
    CONTRACT_STORAGE_ACCESS_KEY?: string
    CONTRACT_STORAGE_SECRET?: string
}

interface TokenizedProps extends cdk.StackProps {
    appConfig: AppConfig;
    ec2NATInstanceClass: ec2.InstanceClass;
    ec2NATInstanceSize: ec2.InstanceSize;
    ec2InstanceClass: ec2.InstanceClass;
    ec2InstanceSize: ec2.InstanceSize;
    enableSSH?: boolean;
}

const app = new cdk.App();

new TokenizedEC2Stack(app, 'TokenizedEC2', {
    appConfig: {
        // TODO: Configure this appropriately for your environment
        // OPERATOR_NAME: "Tokenized EC2",
        // VERSION: "0.1.0",
        // PRIV_KEY: "yourprivatekey",
        // FEE_ADDRESS: "yourfeeaddress",
        // BITCOIN_CHAIN: "mainnet",
        // START_HASH: "000000000000000005f24ad5d65547ad5b05cf61bb30279a47c0aa6c51f4f63e",
        // NODE_ADDRESS: "127.0.0.1:8333",
        // NODE_USER_AGENT: "Tokenized",
        // RPC_HOST: "127.0.0.1:8332",
        // RPC_USERNAME: "youruser",
        // RPC_PASSWORD: "yoursecretpassword",
    },
    ec2NATInstanceClass: ec2.InstanceClass.T3,
    ec2NATInstanceSize: ec2.InstanceSize.Micro,
    ec2InstanceClass: ec2.InstanceClass.T3,
    ec2InstanceSize: ec2.InstanceSize.Micro,
    enableSSH: false,
});

app.run();
