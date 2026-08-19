import { type ReactNode, useState } from "react";
import { ConfigProvider, Switch } from "antd";

import { type CanvasTheme } from "@/lib/canvas-theme";
import {
    imageAspectAllowed,
    imageAspectForSize,
    imageAspectOptions,
    imageQualityLabel as formatImageQualityLabel,
    imageQualityOptions,
    imageResolutionConfigChange,
    imageResolutionOptionForValue,
    imageResolutionOptions,
    imageSizeForAspect,
    imageSizeLabel as formatImageSizeLabel,
    type ImageAspectOption,
    type ImageResolutionOption,
} from "@/lib/image-generation-options";
import { modelCapabilityConfigFor, normalizeImageValue } from "@/lib/model-capabilities";
import { type AiConfig } from "@/stores/use-config-store";

const DIMENSION_STEP = 16;

type ImageSettingsPanelProps = {
    config: AiConfig;
    onConfigChange: (key: "quality" | "size" | "transparentBackground" | "count", value: string) => void;
    theme: CanvasTheme;
    showTitle?: boolean;
    showCount?: boolean;
    className?: string;
    maxCount?: number;
    quickCount?: number;
};

export function ImageSettingsPanel({ config, onConfigChange, theme, showTitle = true, showCount = true, className = "w-[304px] space-y-3 rounded-2xl px-1 py-0.5", maxCount = 15, quickCount = 3 }: ImageSettingsPanelProps) {
    const [snapDimensionToStep, setSnapDimensionToStep] = useState(true);
    const profile = modelCapabilityConfigFor(config, config.model || config.imageModel).image!;
    const normalized = normalizeImageValue(profile, config);
    const quality = normalized.quality;
    const transparentBackground = normalized.transparentBackground === "true";
    const effectiveMaxCount = Math.min(maxCount, profile.maxOutputs);
    const count = Math.max(1, Math.min(effectiveMaxCount, Number(normalized.count)));
    const activeSize = normalized.size;
    const availableAspects = imageAspectOptions.filter((item) => imageAspectAllowed(profile, item));
    const selectedAspect = imageAspectForSize(activeSize) || availableAspects.find((item) => imageSizeForAspect(profile, item) === activeSize) || availableAspects[0];
    const selectedResolution = imageResolutionOptionForValue(profile, normalized, selectedAspect);
    const resolutionOptions = imageResolutionOptions(profile, selectedAspect);
    const qualityOptions = imageQualityOptions(profile);
    const dimensions = readSizeDimensions(activeSize, selectedAspect || imageAspectOptions[0]);
    const selectAspect = (option: ImageAspectOption) => {
        const nextResolutionOptions = imageResolutionOptions(profile, option);
        const nextResolution = selectedResolution ? nextResolutionOptions.find((item) => item.source === selectedResolution.source && item.resolution === selectedResolution.resolution) || nextResolutionOptions[0] : undefined;
        const nextSize = nextResolution?.source === "size" ? nextResolution.size : imageSizeForAspect(profile, option);
        if (nextSize !== activeSize) onConfigChange("size", nextSize);
        if (nextResolution?.source === "quality" && nextResolution.quality !== quality) onConfigChange("quality", nextResolution.quality);
    };
    const selectResolution = (option: ImageResolutionOption) => {
        const change = imageResolutionConfigChange(option);
        onConfigChange(change.key, change.value);
    };
    const updateDimension = (key: "width" | "height", value: number | null) => {
        const next = Math.max(1, Math.floor(value || dimensions[key] || 1024));
        const width = key === "width" ? next : dimensions.width;
        const height = key === "height" ? next : dimensions.height;
        onConfigChange("size", `${alignDimension(width, snapDimensionToStep)}x${alignDimension(height, snapDimensionToStep)}`);
    };

    return (
        <ImageSettingsTheme theme={theme}>
            <div
                className={className}
                style={{ color: theme.node.text }}
                onMouseDown={(event) => {
                    event.stopPropagation();
                    if (event.target instanceof HTMLInputElement) return;
                    if (document.activeElement instanceof HTMLInputElement && event.currentTarget.contains(document.activeElement)) document.activeElement.blur();
                }}
            >
                {showTitle ? <div className="text-base font-semibold">图像设置</div> : null}
                {availableAspects.length ? (
                    <div className="space-y-2">
                        <SettingTitle color={theme.node.muted}>比例</SettingTitle>
                        <div className="grid grid-cols-4 gap-1.5 sm:grid-cols-7">
                            {availableAspects.map((item) => (
                                <button
                                    key={item.value}
                                    type="button"
                                    className="flex h-[52px] cursor-pointer flex-col items-center justify-center gap-0.5 rounded-lg bg-transparent text-[var(--fs-label)] transition-colors hover:brightness-110 focus-visible:outline focus-visible:outline-1 focus-visible:outline-offset-1"
                                    style={{ background: selectedAspect?.value === item.value ? theme.toolbar.activeBg : "transparent", color: theme.node.text, outlineColor: theme.node.muted }}
                                    onMouseDown={(event) => event.stopPropagation()}
                                    onClick={() => selectAspect(item)}
                                >
                                    <AspectIcon type={item.icon} width={item.width} height={item.height} color={theme.node.text} />
                                    <span className="whitespace-nowrap">{item.label}</span>
                                </button>
                            ))}
                        </div>
                    </div>
                ) : null}
                {resolutionOptions.length ? (
                    <div className="space-y-2">
                        <SettingTitle color={theme.node.muted}>清晰度</SettingTitle>
                        <div className={`grid gap-1.5 ${resolutionOptions.length <= 2 ? "grid-cols-2" : "grid-cols-3"}`}>
                            {resolutionOptions.map((item) => (
                                <OptionPill key={item.value} selected={selectedResolution?.value === item.value} theme={theme} onClick={() => selectResolution(item)}>
                                    {item.label}
                                </OptionPill>
                            ))}
                        </div>
                    </div>
                ) : null}
                {qualityOptions.length > 1 ? (
                    <div className="space-y-2">
                        <SettingTitle color={theme.node.muted}>质量</SettingTitle>
                        <div className={`grid gap-1.5 ${qualityOptions.length <= 2 ? "grid-cols-2" : "grid-cols-4"}`}>
                            {qualityOptions.map((item) => (
                                <OptionPill key={item.value} selected={quality === item.value} theme={theme} onClick={() => onConfigChange("quality", item.value)}>
                                    {item.label}
                                </OptionPill>
                            ))}
                        </div>
                    </div>
                ) : null}
                {profile.size.parameter !== "none" && profile.size.allowCustom ? (
                    <div className="space-y-2">
                        <div className="flex items-center justify-between gap-3">
                            <SettingTitle color={theme.node.muted}>自定义尺寸</SettingTitle>
                            <div className="flex items-center gap-2">
                                <span className="text-xs font-medium" style={{ color: theme.node.muted }}>
                                    16倍数对齐
                                </span>
                                <span title="输入完成后自动向上补成 16 的倍数" onMouseDown={(event) => event.stopPropagation()}>
                                    <Switch size="small" checked={snapDimensionToStep} onChange={setSnapDimensionToStep} />
                                </span>
                            </div>
                        </div>
                        <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-1.5">
                            <DimensionInput prefix="W" value={dimensions.width} disabled={activeSize === "auto"} theme={theme} alignToStep={snapDimensionToStep} onChange={(value) => updateDimension("width", value)} />
                            <span className="text-sm opacity-45">↔</span>
                            <DimensionInput prefix="H" value={dimensions.height} disabled={activeSize === "auto"} theme={theme} alignToStep={snapDimensionToStep} onChange={(value) => updateDimension("height", value)} />
                        </div>
                    </div>
                ) : null}
                {profile.transparentBackground.supported ? (
                    <div className="flex items-center justify-between gap-3">
                        <div className="min-w-0">
                            <SettingTitle color={theme.node.muted}>透明背景</SettingTitle>
                            <div className="mt-1 text-[var(--fs-label)]" style={{ color: theme.node.muted }}>
                                请求模型输出保留 Alpha 通道的 PNG
                            </div>
                        </div>
                        <span title="是否支持透明背景由当前模型接口决定" onMouseDown={(event) => event.stopPropagation()}>
                            <Switch size="small" checked={transparentBackground} onChange={(checked) => onConfigChange("transparentBackground", checked ? "true" : "false")} />
                        </span>
                    </div>
                ) : null}
                {showCount && effectiveMaxCount > 1 ? (
                    <div className="space-y-2">
                        <SettingTitle color={theme.node.muted}>生成张数</SettingTitle>
                        <div className="grid grid-cols-4 gap-1.5">
                            {Array.from({ length: Math.min(quickCount, effectiveMaxCount) }, (_, index) => index + 1).map((value) => (
                                <OptionPill key={value} selected={count === value} theme={theme} onClick={() => onConfigChange("count", String(value))}>
                                    {value}
                                </OptionPill>
                            ))}
                            <CountInput value={count} quickCount={quickCount} max={effectiveMaxCount} theme={theme} onChange={(value) => onConfigChange("count", String(value || 1))} />
                        </div>
                    </div>
                ) : null}
            </div>
        </ImageSettingsTheme>
    );
}

export function ImageSettingsTheme({ theme, children }: { theme: CanvasTheme; children: ReactNode }) {
    return (
        <ConfigProvider
            theme={{
                token: { colorBgContainer: theme.canvas.background, colorBgElevated: theme.canvas.background, colorBorder: theme.node.stroke, colorPrimary: theme.node.activeStroke, colorText: theme.node.text, colorTextLightSolid: theme.node.panel },
                components: { Button: { defaultBg: theme.canvas.background, defaultBorderColor: theme.node.stroke, defaultColor: theme.node.text } },
            }}
        >
            {children}
        </ConfigProvider>
    );
}

// 保留原有导出，供画布外的设置入口继续复用。
export function imageQualityLabel(value: string) {
    return formatImageQualityLabel(value);
}

export function imageSizeLabel(size: string) {
    return formatImageSizeLabel(size);
}

function OptionPill({ selected, theme, onClick, children }: { selected: boolean; theme: CanvasTheme; onClick: () => void; children: ReactNode }) {
    return (
        <button
            type="button"
            className="h-8 cursor-pointer rounded-full px-2 text-xs transition-colors hover:brightness-110 focus-visible:outline focus-visible:outline-1 focus-visible:outline-offset-1"
            style={{ background: selected ? theme.toolbar.activeBg : "transparent", color: theme.node.text, outlineColor: theme.node.muted }}
            onMouseDown={(event) => event.stopPropagation()}
            onClick={onClick}
        >
            {children}
        </button>
    );
}

function DimensionInput({ prefix, value, disabled, theme, alignToStep, onChange }: { prefix: string; value: number; disabled: boolean; theme: CanvasTheme; alignToStep: boolean; onChange: (value: number | null) => void }) {
    const commit = (input: HTMLInputElement) => {
        const next = alignDimension(Math.max(1, Math.floor(Number(input.value) || value || 1024)), alignToStep);
        input.value = String(next);
        onChange(next);
    };

    return (
        <label className="flex h-8 overflow-hidden rounded-lg text-xs" style={{ background: theme.toolbar.itemHover, color: theme.node.text, opacity: disabled ? 0.55 : 1 }}>
            <span className="grid w-8 place-items-center" style={{ color: theme.node.muted }}>
                {prefix}
            </span>
            <input
                type="number"
                min={1}
                disabled={disabled}
                className="min-w-0 flex-1 bg-transparent px-2 outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                defaultValue={value || ""}
                key={`${prefix}-${value}`}
                onBlur={(event) => commit(event.currentTarget)}
                onKeyDown={(event) => {
                    if (event.key === "Enter") event.currentTarget.blur();
                }}
                onMouseDown={(event) => event.stopPropagation()}
            />
        </label>
    );
}

function CountInput({ value, quickCount, max, theme, onChange }: { value: number; quickCount: number; max: number; theme: CanvasTheme; onChange: (value: number | null) => void }) {
    const commit = (input: HTMLInputElement) => {
        const next = Math.max(1, Math.min(max, Math.floor(Number(input.value) || 1)));
        input.value = String(next);
        onChange(next);
    };
    return (
        <label className="flex h-8 overflow-hidden rounded-full text-xs" style={{ background: theme.toolbar.itemHover, color: theme.node.text }}>
            <input
                key={value > quickCount ? `custom-${value}` : "quick"}
                type="number"
                min={1}
                max={max}
                aria-label="自定义生成张数"
                placeholder="输入"
                className="min-w-0 flex-1 bg-transparent px-2 text-center outline-none placeholder:text-current placeholder:opacity-55 [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                style={{ color: theme.node.text, WebkitTextFillColor: theme.node.text }}
                defaultValue={value > quickCount ? value : ""}
                onBlur={(event) => commit(event.currentTarget)}
                onKeyDown={(event) => {
                    if (event.key === "Enter") event.currentTarget.blur();
                }}
                onMouseDown={(event) => event.stopPropagation()}
            />
        </label>
    );
}

function AspectIcon({ type, width, height, color }: { type: string; width: number; height: number; color: string }) {
    if (type === "auto") return null;
    const ratio = width / Math.max(1, height);
    const boxWidth = ratio >= 1 ? 22 : Math.max(9, 22 * ratio);
    const boxHeight = ratio >= 1 ? Math.max(9, 22 / ratio) : 22;
    return (
        <span className="grid h-6 w-8 place-items-center">
            <span className="border-2" style={{ width: boxWidth, height: boxHeight, borderColor: color }} />
        </span>
    );
}

function SettingTitle({ children, color }: { children: string; color: string }) {
    return (
        <div className="text-xs font-medium" style={{ color }}>
            {children}
        </div>
    );
}

function readSizeDimensions(size: string, fallback: { width: number; height: number }) {
    const match = size?.match(/^(\d+)x(\d+)$/);
    return {
        width: match ? Number(match[1]) : fallback.width,
        height: match ? Number(match[2]) : fallback.height,
    };
}

function alignDimension(value: number, enabled: boolean) {
    return enabled ? Math.ceil(value / DIMENSION_STEP) * DIMENSION_STEP : value;
}
