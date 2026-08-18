import { describe, expect, test } from "bun:test";

import {
    appendProviderEvaluationAttempt,
    createProviderEvaluationPlan,
    evaluationSettingsFingerprint,
    recordBlindPreference,
    recordProviderEvaluationDecision,
    recordProviderEvaluationScore,
    summarizeProviderEvaluation,
    type BlindPreferenceRecord,
    type ProviderEvaluationAttempt,
    type ProviderEvaluationDimensions,
    type ProviderEvaluationPlan,
} from "../src/ecommerce";

const dimensions: ProviderEvaluationDimensions = {
    referenceFidelity: 4,
    productIdentity: 5,
    commercialQuality: 4,
    instructionFollowing: 4,
    variationAbility: 3,
    physicalPlausibility: 4,
    aiArtifactSeverity: 2,
    apiStability: 4,
};

function createPlan(): ProviderEvaluationPlan {
    return createProviderEvaluationPlan({
        planId: "provider-eval-01",
        projectId: "ecommerce-project-01",
        skillRef: "still-life.lifestyle-tabletop@1",
        settings: {
            mode: "STILL_LIFE",
            width: 1024,
            height: 1024,
            quality: "standard",
            outputCount: 4,
            referenceAssetIds: ["asset-mug-01"],
            promptTemplateVersion: "prototype-a@1",
            seedPolicy: "provider_default",
        },
        cases: [
            {
                caseId: "case-01",
                projectId: "ecommerce-project-01",
                fixtureRevision: "fixture@1",
                sourceAssetIds: ["asset-mug-01"],
                productImageId: "asset-mug-01",
                userGoal: "premium lifestyle listing image",
                constraints: ["preserve product identity"],
                createdAt: "2026-08-13T01:00:00Z",
            },
        ],
        candidates: [
            {
                candidateId: "candidate-a",
                adapterId: "adapter-a",
                modelRef: "model-ref-a",
                displayName: "Candidate A",
                availability: "available",
                capabilities: {
                    referenceImage: true,
                    imageEdit: true,
                    asyncJob: true,
                    polling: true,
                    callback: false,
                    costReporting: true,
                },
            },
            {
                candidateId: "candidate-b",
                adapterId: "adapter-b",
                modelRef: "model-ref-b",
                displayName: "Candidate B",
                availability: "unknown",
                capabilities: {
                    referenceImage: true,
                    imageEdit: false,
                    asyncJob: false,
                    polling: false,
                    callback: false,
                    costReporting: false,
                },
            },
        ],
        createdAt: "2026-08-13T01:00:00Z",
    });
}

function attempt(plan: ProviderEvaluationPlan, patch: Partial<ProviderEvaluationAttempt> = {}): ProviderEvaluationAttempt {
    return {
        attemptId: "attempt-a-01",
        planId: plan.planId,
        caseId: "case-01",
        candidateId: "candidate-a",
        variant: "skill",
        status: "succeeded",
        settingsFingerprint: evaluationSettingsFingerprint(plan.settings),
        requestFingerprint: "request-fingerprint-01",
        resultRefs: ["result-a-01"],
        startedAt: "2026-08-13T01:01:00Z",
        completedAt: "2026-08-13T01:01:04Z",
        latencyMs: 4000,
        usage: { outputUnits: 4, unitName: "image" },
        cost: { amount: 0.12, currency: "USD", status: "recorded" },
        evidence: "recorded",
        ...patch,
    };
}

describe("Ecommerce Provider Evaluation contract", () => {
    test("创建计划时只建立可审计骨架，不自动推断 GO", () => {
        const plan = createPlan();
        const summary = summarizeProviderEvaluation(plan);

        expect(plan.schemaVersion).toBe(1);
        expect(plan.attempts).toHaveLength(0);
        expect(summary.decision).toBe("pending");
        expect(summary.hasRecordedResults).toBe(false);
        expect(summary.hasUnknowns).toBe(true);
        expect(summary.candidateSummaries[0].evidence).toBe("unknown");
    });

    test("尝试必须使用计划设置指纹，成功结果才能进入评分", () => {
        const plan = createPlan();
        const recorded = appendProviderEvaluationAttempt(plan, attempt(plan));
        const scored = recordProviderEvaluationScore(recorded, {
            scoreId: "score-a-01",
            attemptId: "attempt-a-01",
            evaluatorRef: "reviewer-01",
            dimensions,
            evidence: "recorded",
            recordedAt: "2026-08-13T01:02:00Z",
        });
        const summary = summarizeProviderEvaluation(scored);

        expect(summary.candidateSummaries[0]).toMatchObject({
            attempts: 1,
            succeeded: 1,
            successRate: 1,
            averageLatencyMs: 4000,
            knownCost: { amount: 0.12, currency: "USD" },
            scoreCount: 1,
        });
        expect(summary.candidateSummaries[0].dimensionAverages.productIdentity).toBe(5);
        expect(summary.candidateSummaries[0].dimensionAverages.apiStability).toBe(4);
        expect(() => appendProviderEvaluationAttempt(plan, attempt(plan, { settingsFingerprint: "different-settings" }))).toThrow("settingsFingerprint");
        expect(() => recordProviderEvaluationScore(plan, {
            scoreId: "score-failed",
            attemptId: "missing",
            evaluatorRef: "reviewer-01",
            dimensions,
            evidence: "recorded",
            recordedAt: "2026-08-13T01:02:00Z",
        })).toThrow("unknown attempt");
    });

    test("失败尝试必须保留结构化失败原因，不能伪装成成功", () => {
        const plan = createPlan();
        expect(() => appendProviderEvaluationAttempt(plan, attempt(plan, { resultRefs: [], status: "failed" }))).toThrow("requires failure");

        const failed = appendProviderEvaluationAttempt(plan, attempt(plan, {
            attemptId: "attempt-b-01",
            candidateId: "candidate-b",
            status: "failed",
            resultRefs: [],
            failure: { code: "provider_timeout", stage: "poll", retryable: true, summary: "polling timed out" },
            evidence: "recorded",
            cost: { status: "unavailable" },
        }));
        const summary = summarizeProviderEvaluation(failed);

        expect(summary.candidateSummaries[1]).toMatchObject({ attempts: 1, failed: 1, successRate: 0, evidence: "recorded" });
        expect(summary.hasUnknowns).toBe(true);
        expect(summary.candidateSummaries[1].failureModes).toEqual({ provider_timeout: 1 });
    });

    test("盲测只能引用同一 case 的已记录成功结果，并保持 blinded 记录", () => {
        let plan = createPlan();
        plan = appendProviderEvaluationAttempt(plan, attempt(plan));
        plan = appendProviderEvaluationAttempt(plan, attempt(plan, {
            attemptId: "attempt-a-02",
            resultRefs: ["result-a-02"],
            requestFingerprint: "request-fingerprint-02",
        }));

        const preference: BlindPreferenceRecord = {
            preferenceId: "blind-01",
            caseId: "case-01",
            resultRefs: ["result-a-01", "result-a-02"],
            preferredResultRef: "result-a-02",
            tie: false,
            voterRef: "human-reviewer-01",
            blinded: true,
            evidence: "recorded",
            recordedAt: "2026-08-13T01:03:00Z",
        };

        expect(() => recordBlindPreference(plan, { ...preference, resultRefs: ["result-a-01", "unknown-result"] })).toThrow("succeeded results");
        expect(() => recordBlindPreference(plan, { ...preference, preferredResultRef: undefined, tie: false })).toThrow("preferred");
        plan = recordBlindPreference(plan, preference);

        expect(plan.blindPreferences).toHaveLength(1);
        expect(plan.blindPreferences[0].blinded).toBe(true);
        expect(summarizeProviderEvaluation(plan).blindPreferenceCount).toBe(1);
    });

    test("GO/MODIFY/STOP 只能由显式决策记录产生", () => {
        let plan = createPlan();
        plan = appendProviderEvaluationAttempt(plan, attempt(plan));
        expect(summarizeProviderEvaluation(plan).decision).toBe("pending");

        plan = recordProviderEvaluationDecision(plan, {
            decision: "modify",
            reviewerRef: "human-reviewer-01",
            rationale: "reference fidelity is promising but the product identity needs another pass",
            nextAction: "adjust skill constraints and repeat the same fixture",
            evidence: "recorded",
            recordedAt: "2026-08-13T01:04:00Z",
        });

        expect(summarizeProviderEvaluation(plan).decision).toBe("modify");
        expect(summarizeProviderEvaluation(plan).hasRecordedResults).toBe(true);
    });
});
