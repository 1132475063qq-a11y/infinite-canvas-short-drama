import { describe, expect, test } from "bun:test";

import {
    createModelChannel,
    effectiveConfigForChannelPolicy,
    encodeChannelModel,
    normalizeConfigSnapshot,
} from "../src/stores/use-config-store";

describe("platform channel policy", () => {
    const systemChannel = createModelChannel({
        id: "system-image",
        name: "平台图片模型",
        scope: "system",
        apiKey: "system",
        models: ["gpt-image-2"],
        modelCosts: [{
            model: "gpt-image-2",
            capability: "image",
            protocol: "openai-image",
            billingMode: "fixed_request",
            unitPriceMicrocredits: 1_000_000,
        }],
    });
    const userChannel = createModelChannel({
        id: "user-bypass",
        name: "用户自带渠道",
        apiKey: "user-secret",
        models: ["private-image-model"],
    });
    const config = normalizeConfigSnapshot({
        config: {
            channels: [systemChannel, userChannel],
            imageModel: encodeChannelModel(userChannel.id, "private-image-model"),
        },
    }).config;

    test("platform mode exposes only administrator-managed system channels", () => {
        const effective = effectiveConfigForChannelPolicy(config, false);

        expect(effective.channels.map((channel) => channel.id)).toEqual([systemChannel.id]);
        expect(effective.models).toEqual([encodeChannelModel(systemChannel.id, "gpt-image-2")]);
        expect(effective.imageModel).toBe(encodeChannelModel(systemChannel.id, "gpt-image-2"));
        expect(JSON.stringify(effective)).not.toContain("user-secret");
    });

    test("an explicit administrator switch can restore BYOK for private deployments", () => {
        const effective = effectiveConfigForChannelPolicy(config, true);

        expect(effective.channels.map((channel) => channel.id)).toEqual([systemChannel.id, userChannel.id]);
        expect(effective.imageModel).toBe(encodeChannelModel(userChannel.id, "private-image-model"));
    });
});
