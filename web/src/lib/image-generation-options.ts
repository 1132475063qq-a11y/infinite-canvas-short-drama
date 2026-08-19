import type { ImageCapabilityConfig } from "@/lib/model-capabilities";

export type ImageAspectOption = {
    value: string;
    label: string;
    width: number;
    height: number;
    icon: "square" | "landscape" | "portrait";
};

export type ImageResolutionOption = {
    value: string;
    label: string;
};

// 界面先选比例，再通过清晰度选择模型对应的像素尺寸。
export const imageAspectOptions: ImageAspectOption[] = [
    { value: "1:1", label: "1:1", width: 1024, height: 1024, icon: "square" },
    { value: "4:3", label: "4:3", width: 1360, height: 1024, icon: "landscape" },
    { value: "3:4", label: "3:4", width: 1024, height: 1360, icon: "portrait" },
    { value: "16:9", label: "16:9", width: 1824, height: 1024, icon: "landscape" },
    { value: "9:16", label: "9:16", width: 1024, height: 1824, icon: "portrait" },
    { value: "3:2", label: "3:2", width: 1536, height: 1024, icon: "landscape" },
    { value: "2:3", label: "2:3", width: 1024, height: 1536, icon: "portrait" },
];

const qualityResolutionAliases: Record<string, string> = {
    low: "1k",
    standard: "1k",
    medium: "2k",
    hd: "2k",
    high: "4k",
};

export function imageAspectAllowed(profile: ImageCapabilityConfig, aspect: ImageAspectOption) {
    if (profile.size.parameter === "none") return false;
    if (profile.size.allowCustom && profile.size.values.length === 0) return true;
    return profile.size.values.some((value) => matchesAspect(aspect, value));
}

export function imageAspectForSize(value: string | undefined) {
    const normalized = normalizeSizeValue(value);
    if (!normalized || normalized === "auto") return undefined;
    return imageAspectOptions.find((aspect) => normalizeSizeValue(aspect.value) === normalized) || imageAspectOptions.find((aspect) => matchesAspect(aspect, normalized));
}

export function imageSizeForAspect(profile: ImageCapabilityConfig, aspect: ImageAspectOption, resolution?: string) {
    if (profile.size.parameter === "none") return "auto";
    const candidates = profile.size.values.filter((value) => matchesAspect(aspect, value));
    const requestedResolution = normalizeResolution(resolution);
    if (requestedResolution) {
        const resolutionMatch = candidates.find((value) => imageResolutionFromSize(value) === requestedResolution);
        if (resolutionMatch) return resolutionMatch;
    }
    const directMatch = candidates.find((value) => normalizeSizeValue(value) === normalizeSizeValue(aspect.value));
    if (directMatch) return directMatch;
    return candidates[0] || aspect.value;
}

export function imageResolutionOptions(profile: ImageCapabilityConfig, aspect?: ImageAspectOption): ImageResolutionOption[] {
    const resolutions = new Set<string>();
    const sizeValues = aspect ? profile.size.values.filter((value) => matchesAspect(aspect, value)) : profile.size.values;

    if (aspect && supportsBaseAspect(profile, aspect) && profile.quality.supported) resolutions.add("1k");
    sizeValues.forEach((value) => {
        const resolution = imageResolutionFromSize(value);
        if (resolution) resolutions.add(resolution);
    });
    if (profile.quality.supported) {
        profile.quality.values.forEach((value) => {
            const resolution = imageResolutionFromQuality(value);
            if (resolution) resolutions.add(resolution);
        });
    }

    return Array.from(resolutions)
        .sort(compareResolutions)
        .map((value) => ({ value, label: imageResolutionLabel(value) }));
}

export function imageResolutionForValue(value: { size?: string; quality?: string }) {
    const sizeResolution = imageResolutionFromSize(value.size);
    if (sizeResolution) return sizeResolution;
    const qualityResolution = imageResolutionFromQuality(value.quality);
    if (qualityResolution) return qualityResolution;
    return imageAspectForSize(value.size) ? "1k" : undefined;
}

export function imageQualityForResolution(profile: ImageCapabilityConfig, resolution: string) {
    if (!profile.quality.supported) return undefined;
    const normalizedResolution = normalizeResolution(resolution);
    if (!normalizedResolution) return undefined;
    const directMatch = profile.quality.values.find((value) => normalizeQualityValue(value) === normalizedResolution);
    if (directMatch) return directMatch;
    const aliasMatch = profile.quality.values.find((value) => imageResolutionFromQuality(value) === normalizedResolution);
    if (aliasMatch) return aliasMatch;
    return profile.quality.values.find((value) => normalizeQualityValue(value) === "auto");
}

export function imageResolutionLabel(value: string | undefined) {
    const normalized = normalizeResolution(value);
    if (!normalized) return "默认";
    return normalized.toUpperCase();
}

export function imageSizeLabel(value: string | undefined) {
    const normalized = normalizeSizeValue(value);
    if (!normalized || normalized === "auto") return "自动";
    return imageAspectForSize(normalized)?.label || value || "自动";
}

function supportsBaseAspect(profile: ImageCapabilityConfig, aspect: ImageAspectOption) {
    if (profile.size.parameter === "none") return false;
    if (profile.size.allowCustom) return true;
    return profile.size.values.some((value) => normalizeSizeValue(value) === normalizeSizeValue(aspect.value));
}

function matchesAspect(aspect: ImageAspectOption, value: string) {
    const candidate = parseAspectValue(value);
    if (!candidate) return false;
    const aspectRatio = aspect.width / aspect.height;
    return Math.abs(candidate.width / candidate.height - aspectRatio) <= 0.02;
}

function parseAspectValue(value: string | undefined) {
    const normalized = normalizeSizeValue(value);
    const ratioMatch = normalized.match(/^(\d+(?:\.\d+)?):(\d+(?:\.\d+)?)/);
    if (ratioMatch) return positiveDimensions(ratioMatch[1], ratioMatch[2]);
    const dimensionsMatch = normalized.match(/^(\d+(?:\.\d+)?)x(\d+(?:\.\d+)?)/);
    if (dimensionsMatch) return positiveDimensions(dimensionsMatch[1], dimensionsMatch[2]);
    return undefined;
}

function positiveDimensions(width: string, height: string) {
    const parsedWidth = Number(width);
    const parsedHeight = Number(height);
    return Number.isFinite(parsedWidth) && Number.isFinite(parsedHeight) && parsedWidth > 0 && parsedHeight > 0 ? { width: parsedWidth, height: parsedHeight } : undefined;
}

function imageResolutionFromSize(value: string | undefined) {
    const normalized = normalizeSizeValue(value);
    const suffixMatch = normalized.match(/-(\d+)k$/);
    if (suffixMatch) return `${Math.max(1, Number(suffixMatch[1]))}k`;
    const dimensions = parseAspectValue(normalized);
    if (!dimensions || !normalized.includes("x")) return undefined;
    return `${Math.max(1, Math.round(Math.max(dimensions.width, dimensions.height) / 1024))}k`;
}

function imageResolutionFromQuality(value: string | undefined) {
    const normalized = normalizeQualityValue(value);
    if (/^\d+k$/.test(normalized)) return normalized;
    return qualityResolutionAliases[normalized];
}

function compareResolutions(left: string, right: string) {
    const leftValue = Number.parseInt(left, 10);
    const rightValue = Number.parseInt(right, 10);
    if (Number.isFinite(leftValue) && Number.isFinite(rightValue) && leftValue !== rightValue) return leftValue - rightValue;
    return left.localeCompare(right);
}

function normalizeSizeValue(value: string | undefined) {
    return (value || "").trim().toLowerCase().replace("×", "x");
}

function normalizeQualityValue(value: string | undefined) {
    return (value || "").trim().toLowerCase();
}

function normalizeResolution(value: string | undefined) {
    const normalized = normalizeQualityValue(value);
    return /^\d+k$/.test(normalized) ? normalized : "";
}
