#!/usr/bin/env node

import cdk = require('@aws-cdk/cdk');
import ec2 = require('@aws-cdk/aws-ec2');
import ecs = require('@aws-cdk/aws-ecs');
import s3 = require('@aws-cdk/aws-s3');

class TokenizedECSStack extends cdk.Stack {
    constructor(parent: cdk.App, name: string, props: TokenizedProps) {
        super(parent, name, props);

        this.addWarning("This stack is a WIP example and may not be 100% production ready. Use at your own risk.");

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ec2.html
        const vpc = new ec2.VpcNetwork(this, 'Tokenized-VPC');

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#clusters
        const cluster = new ecs.Cluster(this, 'Cluster', {
            vpc,
        });

        cluster.addDefaultAutoScalingGroupCapacity({
            instanceType: new ec2.InstanceTypePair(props.ec2InstanceClass, props.ec2InstanceSize),
            instanceCount: 1,
        });

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-s3.html

        // Setup Node Storage
        let nodeStorageBucket;
        if (typeof props.appConfig.NODE_STORAGE_BUCKET == "undefined") {
            nodeStorageBucket = new s3.Bucket(this, "NodeStorageBucket")
        } else if (props.appConfig.NODE_STORAGE_BUCKET.toLowerCase() == "standalone") {
            this.addError("NODE_STORAGE_BUCKET=standalone means that any data will get lost when the container is terminated. This is probably not what you want to happen..");
        }
        else {
            this.addWarning(`The specified NODE_STORAGE_BUCKET (${ props.appConfig.NODE_STORAGE_BUCKET}) is defined externally to this stack. Make sure appropriate permissions are set so we can write to it, or weird things might happen.`);

            nodeStorageBucket = s3.Bucket.import(this, "NodeStorageBucket", {
                bucketName: props.appConfig.NODE_STORAGE_BUCKET
            });
        }

        // Setup Contract Storage
        let contractStorageBucket;
        if (typeof props.appConfig.CONTRACT_STORAGE_BUCKET == "undefined") {
            contractStorageBucket = new s3.Bucket(this, "ContractStorageBucket")
        } else if (props.appConfig.CONTRACT_STORAGE_BUCKET.toLowerCase() == "standalone") {
            this.addError("CONTRACT_STORAGE_BUCKET=standalone means that any data will get lost when the container is terminated. This is probably not what you want to happen..");
        }
        else {
            this.addWarning(`The specified CONTRACT_STORAGE_BUCKET (${ props.appConfig.CONTRACT_STORAGE_BUCKET}) is defined externally to this stack. Make sure appropriate permissions are set so we can write to it, or weird things might happen.`);

            contractStorageBucket = s3.Bucket.import(this, "ContractStorageBucket", {
                bucketName: props.appConfig.CONTRACT_STORAGE_BUCKET
            });
        }

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#task-definitions
        const ec2TaskDefinition = new ecs.Ec2TaskDefinition(this, 'SmartContractTaskDef', {
            networkMode: ecs.NetworkMode.Bridge
        });

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#images
        // TODO: Build image from local sources directly? ecs.ContainerImage.fromAsset(this, 'Image', { directory: './image' })
        const defaultContainerName = "tokenizer/smartcontractd";
        const containerImage = ecs.ContainerImage.fromDockerHub(props.containerName ? props.containerName : defaultContainerName);

        let containerEnv = props.appConfig;

        if (nodeStorageBucket) {
            containerEnv.NODE_STORAGE_BUCKET = nodeStorageBucket.bucketName;
            containerEnv.NODE_STORAGE_REGION = new cdk.AwsRegion().resolve();
        }

        if (contractStorageBucket) {
            containerEnv.CONTRACT_STORAGE_BUCKET = contractStorageBucket.bucketName;
            containerEnv.CONTRACT_STORAGE_REGION = new cdk.AwsRegion().resolve();
        }

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#@aws-cdk/aws-ecs.Ec2TaskDefinition.addContainer
        ec2TaskDefinition.addContainer('SmartcontractdContainer', {
            essential: true,
            image: containerImage,
            // @ts-ignore
            environment: containerEnv, // TODO: Solve this properly: TS2322: Type 'AppConfig' is not assignable to type '{ [key: string]: string; }'. Index signature is missing in type 'AppConfig'.
            memoryLimitMiB: 128,
        });

        if (nodeStorageBucket) {
            nodeStorageBucket.grantReadWrite(ec2TaskDefinition.taskRole);
        }

        if (contractStorageBucket) {
            contractStorageBucket.grantReadWrite(ec2TaskDefinition.taskRole);
        }

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#ec2service
        new ecs.Ec2Service(this, 'SmartcontractdService', {
            cluster,
            taskDefinition: ec2TaskDefinition,
            desiredCount: 1,
        });
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

    FEE_ADDRESS: string
    FEE_VALUE: string

    NODE_ADDRESS: string
    NODE_USER_AGENT: string

    RPC_HOST: string
    RPC_USERNAME: string
    RPC_PASSWORD: string

    PRIV_KEY: string

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
    ec2InstanceClass: ec2.InstanceClass;
    ec2InstanceSize: ec2.InstanceSize;
    containerName?: string;
}

const app = new cdk.App();

new TokenizedECSStack(app, 'TokenizedECS', {
    appConfig: {
        // TODO: Configure this appropriately for your environment
        // OPERATOR_NAME: "Standalone",
        // VERSION: "0.1",
        // FEE_ADDRESS: "yourfeeaddress",
        // FEE_VALUE: "2000",
        // NODE_ADDRESS: "1.2.3.4:8333",
        // NODE_USER_AGENT: "",
        // RPC_HOST: "1.2.3.4:8332",
        // RPC_USERNAME: "youruser",
        // RPC_PASSWORD: "yoursecretpassword",
        // PRIV_KEY: "yourwif",
    },
    ec2InstanceClass: ec2.InstanceClass.T3,
    ec2InstanceSize: ec2.InstanceSize.Micro,
});

app.run();
