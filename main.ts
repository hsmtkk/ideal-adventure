// Copyright (c) HashiCorp, Inc
// SPDX-License-Identifier: MPL-2.0
import { Construct } from "constructs";
import { App, TerraformStack, CloudBackend, NamedCloudWorkspace } from "cdktf";
import * as google from '@cdktf/provider-google';
import * as path from 'path';
import { AssetType, TerraformAsset } from "cdktf/lib/terraform-asset";

const project = 'ideal-adventure';
const region = 'us-central1';

class MyStack extends TerraformStack {
  constructor(scope: Construct, id: string) {
    super(scope, id);

    new google.provider.GoogleProvider(this, 'google', {
      project,
      region,
    });

    new google.storageBucket.StorageBucket(this, 'dataset', {
      location: region,
      name: `dataset-${project}`,
    });

    const assetBucket = new google.storageBucket.StorageBucket(this, 'asset-bucket', {
      location: region,
      name: `asset-${project}`,      
    });

    const sourceBucket = new google.storageBucket.StorageBucket(this, 'source-bucket', {
      location: region,
      name: `source-${project}`,
    });

    const destinationBucket = new google.storageBucket.StorageBucket(this, 'destination-bucket', {
      location: region,
      name: `destination-${project}`,
    });

    const asset = new TerraformAsset(this, 'asset', {
      path: path.resolve('function'),
      type: AssetType.ARCHIVE,
    });

    const assetObject = new google.storageBucketObject.StorageBucketObject(this, 'asset-object', {
      bucket: assetBucket.name,
      name: `${asset.assetHash}.zip`,
      source: asset.path,
    });

    new google.cloudfunctions2Function.Cloudfunctions2Function(this, 'function', {
      buildConfig: {
        entryPoint: 'imageUploaded',
        runtime: 'go119',
        source: {
          storageSource: {
            bucket: assetBucket.name,
            object: assetObject.name,
          },
        },
      },
      eventTrigger: {
        eventType: 'google.cloud.storage.object.v1.finalized',
        eventFilters: [{
          attribute: 'bucket',
          value: sourceBucket.name,
        }],
      },
      location: region,
      name: 'function',
      serviceConfig: {
        environmentVariables: {
          'DESTINATION_BUCKET': destinationBucket.name,
          'PROJECT_ID': project,
          'ENDPOINT_ID': '7212484016908271616',
        },
      },
    });

  }
}

const app = new App();
const stack = new MyStack(app, "ideal-adventure");
new CloudBackend(stack, {
  hostname: "app.terraform.io",
  organization: "hsmtkkdefault",
  workspaces: new NamedCloudWorkspace("ideal-adventure")
});
app.synth();
