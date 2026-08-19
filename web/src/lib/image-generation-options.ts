import type { ImageCapabilityConfig } from "@/lib/model-capabilities";

export type ImageAspectOption = {
    value: string;
    label: string;
    width: number;
    height: number;
    icon: "square" | "landscape" | "portrait";
};

export type ImageResolutionOption = { value: string; label: string; resolution: string; source: "size"; size: string } | { value: string; label: string; resolution: string; source: "quality"; quality: string };

export type ImageQualityOption = {
    value: string;
    label: string;
};

// 比例和清晰度是两个独立概念：固定像素尺寸写入 size，明确的 K 档模型写入 quality。
export const imageAspectOptions: ImageAspectOption[] = [
    { value: "1:1", label: "1:1", width: 1024, height: 1024, icon: "square" },
    { value: "4:3", label: "4:3", width: 1360, height: 1024, icon: "landscape" },
    { value: "3:4", label: "3:4", width: 1024, height: 1360, icon: "portrait" },
    { value: "16:9", label: "16:9", width: 1824, height: 1024, icon: "landscape" },
    { value: "21:9", label: "21:9", width: 2352, height: 1008, icon: "landscape" },
    { value: "9:16", label: "9:16", width: 1024, height: 1824, icon: "portrait" },
    { value: "3:2", label: "3:2", width: 1536, height: 1024, icon: "landscape" },
    { value: "2:3", label: "2:3", width: 1024, height: 1536, icon: "portrait" },
];

const qualityLabels: Record<string, string> = {
    auto: "自动",
    low: "低",
    medium: "中",
    high: "高",
    standard: "标准",
    hd: "高清",
    "1k": "1K",
    "2k": "2K",
    "4k": "4K",
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
    const sizeOptions = imageResolutionOptionsForSize(profile, aspect);
    const qualityOptions = imageResolutionOptionsForQuality(profile);

    // ratio + quality(1k/2k) 的模型必须使用 quality；固定像素模型则优先使用 size。
    if (profile.size.parameter === "aspect_ratio" && qualityOptions.length) return qualityOptions;
    if (profile.size.parameter === "aspect_ratio") return [];
    if (sizeOptions.length) return sizeOptions;
    return qualityOptions;
}

export function imageResolutionOptionForValue(profile: ImageCapabilityConfig, value: { size?: string; quality?: string }, aspect = imageAspectForSize(value.size)) {
    const options = imageResolutionOptions(profile, aspect);
    const sizeMatch = options.find((option) => option.source === "size" && normalizeSizeValue(option.size) === normalizeSizeValue(value.size));
    if (sizeMatch) return sizeMatch;
    return options.find((option) => option.source === "quality" && normalizeQualityValue(option.quality) === normalizeQualityValue(value.quality));
}

export function imageResolutionConfigChange(option: ImageResolutionOption) {
    return option.source === "size" ? { key: "size" as const, value: option.size } : { key: "quality" as const, value: option.quality };
}

export function imageQualityOptions(profile: ImageCapabilityConfig): ImageQualityOption[] {
    if (!profile.quality.supported) return [];
    return profile.quality.values.filter((value) => !/^\d+k$/i.test(value.trim())).map((value) => ({ value, label: imageQualityLabel(value) }));
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
    return profile.quality.values.find((value) => imageResolutionFromQuality(value) === normalizedResolution);
}

export function imageResolutionLabel(value: string | undefined) {
    const normalized = normalizeResolution(value);
    if (!normalized) return "默认";
    return normalized.toUpperCase();
}

export function imageQualityLabel(value: string | undefined) {
    const normalized = normalizeQualityValue(value);
    return qualityLabels[normalized] || value || "默认";
}

export function imageSizeLabel(value: string | undefined) {
    const normalized = normalizeSizeValue(value);
    if (!normalized || normalized === "auto") return "自动";
    return imageAspectForSize(normalized)?.label || value || "自动";
}

function imageResolutionOptionsForSize(profile: ImageCapabilityConfig, aspect?: ImageAspectOption): ImageResolutionOption[] {
    if (profile.size.parameter === "none") return [];
    const candidates = aspect ? profile.size.values.filter((value) => matchesAspect(aspect, value)) : profile.size.values;
    const options = new Map<string, ImageResolutionOption>();

    candidates.forEach((size) => {
        const resolution = imageResolutionFromSize(size);
        if (!resolution) return;
        options.set(resolution, { value: `size:${normalizeSizeValue(size)}`, label: imageResolutionLabel(resolution), resolution, source: "size", size });
    });

    const baseSize = aspect && candidates.find((size) => normalizeSizeValue(size) === normalizeSizeValue(aspect.value));
    if (baseSize && !options.has("1k")) {
        options.set("1k", { value: `size:${normalizeSizeValue(baseSize)}`, label: imageResolutionLabel("1k"), resolution: "1k", source: "size", size: baseSize });
    }

    return sortResolutionOptions([...options.values()]);
}

function imageResolutionOptionsForQuality(profile: ImageCapabilityConfig): ImageResolutionOption[] {
    if (!profile.quality.supported) return [];
    const options = new Map<string, ImageResolutionOption>();
    profile.quality.values.forEach((quality) => {
        const normalized = normalizeQualityValue(quality);
        if (!/^\d+k$/.test(normalized)) return;
        options.set(normalized, { value: `quality:${normalized}`, label: imageResolutionLabel(normalized), resolution: normalized, source: "quality", quality });
    });
    return sortResolutionOptions([...options.values()]);
}

function sortResolutionOptions(options: ImageResolutionOption[]) {
    return options.sort((left, right) => compareResolutions(left.resolution, right.resolution));
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
    return /^\d+k$/.test(normalized) ? normalized : undefined;
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
