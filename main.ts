// Copyright (c) HashiCorp, Inc
// SPDX-License-Identifier: MPL-2.0
import { Construct } from "constructs";
import { App, TerraformStack, CloudBackend, NamedCloudWorkspace } from "cdktf";
import * as google from '@cdktf/provider-google';

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
