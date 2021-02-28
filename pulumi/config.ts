import * as pulumi from "@pulumi/pulumi";
import * as random from "@pulumi/random";

const name = new random.RandomString("resourcegroup-name", {
    length: 6,
    special: false,
    upper: false,
});

export const resourceGroupName = pulumi.interpolate`t-${name.result}`;
