import { describe, expect, test } from "bun:test";

import { STILL_LIFE_LIFESTYLE_TABLETOP, bindExpectedRelationGraph, createEcommerceArtifact, createEcommerceArtifactRevision, createProductDNA, isEcommerceMode, isEcommerceNodeKind, prototypeChainToArtifactInputs, runProviderFreePrototypeA } from "../src/ecommerce";

describe("Ecommerce Prototype A contracts", () => {
    test("保持 ProductDNA 的事实分层并去重源资产", () => {
        const product = createProductDNA({
            sourceAssets: ["asset-01", "asset-01"],
            category: "ceramic mug",
            confirmedFacts: { color: "cream" },
            inferredFacts: { finish: "matte" },
            unknownFacts: ["exact material formula"],
            forbiddenClaims: ["handmade"],
            confidence: 0.8,
        });

        expect(product.sourceAssets).toEqual(["asset-01"]);
        expect(product.confirmedFacts).toEqual({ color: "cream" });
        expect(product.inferredFacts).toEqual({ finish: "matte" });
        expect(product.unknownFacts).toEqual(["exact material formula"]);
        expect(product.forbiddenClaims).toEqual(["handmade"]);
    });

    test("Finalized Artifact 只能通过新 id 创建不可变 revision", () => {
        const first = createEcommerceArtifact({
            id: "dna-v1",
            projectId: "project-01",
            artifactType: "product_dna",
            schemaVersion: 1,
            payload: { confirmed: true },
            sourceRefs: ["asset-01"],
            authorityRefs: ["asset-01"],
            evidence: "recorded",
            now: "2026-08-13T00:00:00Z",
        });
        const finalized = { ...first, lifecycle: "finalized" as const };
        const second = createEcommerceArtifactRevision(finalized, { confirmed: false }, "2026-08-13T00:01:00Z", "dna-v2");

        expect(finalized).toMatchObject({ id: "dna-v1", revision: 1, lifecycle: "finalized", payload: { confirmed: true } });
        expect(second).toMatchObject({ id: "dna-v2", revision: 2, lifecycle: "draft", payload: { confirmed: false } });
        expect(() => createEcommerceArtifactRevision(finalized, {}, "2026-08-13T00:02:00Z", "dna-v1")).toThrow("new id");
    });

    test("ExpectedRelationGraph 只能从 Skill 模板绑定变量", () => {
        const graph = bindExpectedRelationGraph("still-life.lifestyle-tabletop", 1, STILL_LIFE_LIFESTYLE_TABLETOP.relationTemplates, { product: "dna-v1", surface: "stone", props: "botanical", logo: "dna-v1:logo", camera: "three-quarter" });

        expect(graph.relations).toHaveLength(3);
        expect(graph.relations[0]).toMatchObject({ from: "dna-v1", relation: "rests_on", to: "stone", source: "skill_template" });
        expect(() => bindExpectedRelationGraph("skill", 1, [{ from: "{{missing}}", relation: "rests_on", to: "surface" }], {})).toThrow("missing");
    });

    test("Prototype A provider-free 链路生成四个 variations、Result Grid 和 UNCERTAIN QA", () => {
        const chain = runProviderFreePrototypeA({
            projectId: "ecom-project-01",
            productAssetId: "asset-mug-01",
            productDna: {
                category: "ceramic mug",
                color: "cream",
                shape: "rounded cylinder",
                logo: "small black mark",
                mustPreserve: ["cream color", "rounded handle"],
                forbiddenChanges: ["no invented text"],
                confidence: 0.9,
            },
            userGoal: "premium lifestyle listing image",
            now: "2026-08-13T00:00:00Z",
        });

        expect(chain.productDna.sourceRefs).toEqual(["asset-mug-01"]);
        expect(chain.creativeDirection.payload.productDnaArtifactId).toBe(chain.productDna.id);
        expect(chain.scenePlan.payload.creativeDirectionArtifactId).toBe(chain.creativeDirection.id);
        expect(chain.creativeShotPlan.payload.skillRef).toBe("still-life.lifestyle-tabletop@1");
        expect(chain.creativeShotPlan.payload.variations).toHaveLength(4);
        expect(chain.creativeShotPlan.payload.variations.every((variation) => variation.expectedRelations.relations.length === 3)).toBe(true);
        expect(chain.resultGrid.slots).toHaveLength(4);
        expect(chain.resultGrid.slots.every((slot) => slot.status === "waiting")).toBe(true);
        expect(chain.basicQA.outcome).toBe("UNCERTAIN");
        expect(chain.basicQA.needsYou).toBe(true);
    });

    test("Ecommerce 类型和节点枚举不与 Film 枚举混淆", () => {
        expect(isEcommerceMode("STILL_LIFE")).toBe(true);
        expect(isEcommerceMode("scene")).toBe(false);
        expect(isEcommerceNodeKind("product_dna")).toBe(true);
        expect(isEcommerceNodeKind("shot")).toBe(false);
    });

    test("Prototype A 持久化引用使用稳定 artifactKey 而不是临时内存 id", () => {
        const chain = runProviderFreePrototypeA({
            projectId: "ecom-project-01",
            productAssetId: "asset/mug 01",
            productDna: { confidence: 0 },
            now: "2026-08-13T00:00:00Z",
        });

        const inputs = prototypeChainToArtifactInputs(chain);
        expect(inputs).toHaveLength(5);
        expect(inputs[0].artifactKey).toBe("product-dna:asset-mug-01");
        expect(inputs[1].sourceRefs).toEqual(["product-dna:asset-mug-01"]);
        expect(inputs[3].skillRef).toBe("still-life.lifestyle-tabletop@1");
        expect(inputs[4].sourceRefs).toEqual(["creative-shot-plan:asset-mug-01"]);
    });
});
