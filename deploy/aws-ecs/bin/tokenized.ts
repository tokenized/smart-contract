#!/usr/bin/env node

import cdk = require('@aws-cdk/cdk');
import ec2 = require('@aws-cdk/aws-ec2');
import ecs = require('@aws-cdk/aws-ecs');

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

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#task-definitions
        const ec2TaskDefinition = new ecs.Ec2TaskDefinition(this, 'SmartContractTaskDef', {
            networkMode: ecs.NetworkMode.Bridge
        });

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#images
        // TODO: Build image from local sources directly? ecs.ContainerImage.fromAsset(this, 'Image', { directory: './image' })
        const containerImage = ecs.ContainerImage.fromDockerHub("tokenizer/smartcontract");

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#@aws-cdk/aws-ecs.Ec2TaskDefinition.addContainer
        ec2TaskDefinition.addContainer('SmartContractContainer', {
            essential: true,
            image: containerImage,
            // @ts-ignore
            environment: props.appConfig, // TODO: Solve this properly: TS2322: Type 'AppConfig' is not assignable to type '{ [key: string]: string; }'. Index signature is missing in type 'AppConfig'.
            memoryLimitMiB: 128,
        });

        // Ref: https://awslabs.github.io/aws-cdk/refs/_aws-cdk_aws-ecs.html#ec2service
        new ecs.Ec2Service(this, 'SmartContractService', {
            cluster,
            taskDefinition: ec2TaskDefinition,
            desiredCount: 1,
        });
    }
}

interface AppConfig {
    TOKENIZER_NAME: string,
    RPC_HOST: string,
    RPC_USERNAME: string,
    RPC_PASSWORD: string,
    FEE_ADDRESS: string,
    FEE: string,
    WIF: string,
    NODE_ADDRESS: string,
    NODE_STORAGE_BUCKET?: string,
    NODE_STORAGE_ROOT: string,
    NODE_STORAGE_REGION?: string,
    NODE_STORAGE_ACCESS_KEY?: string
    NODE_STORAGE_SECRET?: string
}

interface TokenizedProps extends cdk.StackProps {
    appConfig: AppConfig;
    ec2InstanceClass: ec2.InstanceClass;
    ec2InstanceSize: ec2.InstanceSize;
}

const app = new cdk.App();

new TokenizedECSStack(app, 'TokenizedECSStack', {
    appConfig: {
        // TODO: Configure this appropriately for your environment
        // "NODE_ADDRESS": "1.2.3.4:8333",
        // "TOKENIZER_NAME": "Standalone",
        // "RPC_HOST": "1.2.3.4:8332",
        // "RPC_USERNAME": "youruser",
        // "RPC_PASSWORD": "yoursecretpassword",
        // "FEE_ADDRESS": "yourfeeaddress",
        // "FEE": "2000",
        // "WIF": "yourwif",
        // "NODE_STORAGE_ROOT": "",
        // "NODE_STORAGE_BUCKET": "",
        // "NODE_STORAGE_REGION": "",
        // "NODE_STORAGE_ACCESS_KEY": "",
        // "NODE_STORAGE_SECRET": "",
    },
    ec2InstanceClass: ec2.InstanceClass.T3,
    ec2InstanceSize: ec2.InstanceSize.Micro,
});

app.run();
