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

    const functionAccount = new google.serviceAccount.ServiceAccount(this, 'function-account', {
      accountId: 'function-account',
    });

    new google.projectIamBinding.ProjectIamBinding(this, 'function-account-ai', {
      members: [`serviceAccount:${functionAccount.email}`],
      project,
      role: 'roles/aiplatform.user',
    });

    new google.projectIamBinding.ProjectIamBinding(this, 'function-account-storage', {
      members: [`serviceAccount:${functionAccount.email}`],
      project,
      role: 'roles/storage.objectAdmin',
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
        minInstanceCount: 0,
        maxInstanceCount: 1,
        serviceAccountEmail: functionAccount.email,
      },
    });

/*
data "google_storage_project_service_account" "gcs_account" {
}

# To use GCS CloudEvent triggers, the GCS service account requires the Pub/Sub Publisher(roles/pubsub.publisher) IAM role in the specified project.
# (See https://cloud.google.com/eventarc/docs/run/quickstart-storage#before-you-begin)
resource "google_project_iam_member" "gcs-pubsub-publishing" {
  project = "my-project-name"
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"
}
*/

    const storageAccount = new google.dataGoogleStorageProjectServiceAccount.DataGoogleStorageProjectServiceAccount(this, 'storage-account');

    new google.projectIamMember.ProjectIamMember(this, 'storage-account-pubsub', {
      member: `serviceAccount:${storageAccount.emailAddress}`,
      project,
      role: 'roles/pubsub.publisher',
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
