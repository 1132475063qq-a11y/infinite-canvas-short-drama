package service

import "testing"

func TestValidateEcommerceArtifactContract(t *testing.T) {
	for _, artifactType := range []string{EcommerceArtifactTypeProductDNA, EcommerceArtifactTypeCreativeDirection, EcommerceArtifactTypeScenePlan, EcommerceArtifactTypeCreativeShotPlan} {
		if !validEcommerceArtifactType(artifactType) {
			t.Fatalf("expected prototype artifact type to be valid: %s", artifactType)
		}
	}
	if validEcommerceArtifactType("acting") {
		t.Fatal("film artifact type must not enter ecommerce contract")
	}
	if !validEcommerceLifecycle("finalized") || validEcommerceLifecycle("immutable") {
		t.Fatal("unexpected ecommerce lifecycle validation")
	}
	if !validEcommerceEvidence("recorded") || validEcommerceEvidence("guess") {
		t.Fatal("unexpected ecommerce evidence validation")
	}
}

func TestNormalizeEcommerceRefsDeduplicatesAndTrims(t *testing.T) {
	got := normalizeEcommerceRefs([]string{" asset-1 ", "asset-1", "", "asset-2"})
	if len(got) != 2 || got[0] != "asset-1" || got[1] != "asset-2" {
		t.Fatalf("unexpected refs: %#v", got)
	}
}

func TestNormalizeProjectTypeKeepsOnlyExplicitDomains(t *testing.T) {
	if got, err := normalizeProjectType(""); err != nil || got != "short-drama" {
		t.Fatalf("unexpected default project type: %q, %v", got, err)
	}
	if got, err := normalizeProjectType("ecommerce"); err != nil || got != "ecommerce" {
		t.Fatalf("unexpected ecommerce project type: %q, %v", got, err)
	}
	if _, err := normalizeProjectType("film-and-novel"); err == nil {
		t.Fatal("unknown project types must be rejected")
	}
}
