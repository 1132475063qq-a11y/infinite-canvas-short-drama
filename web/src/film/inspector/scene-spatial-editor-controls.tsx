import type { ReactNode } from "react";
import { Button } from "antd";
import { Plus, Trash2 } from "lucide-react";

import type { SpatialPointDraft } from "./scene-spatial-editor-model";

const inputClassName = "w-full rounded-md border border-border/80 bg-foreground/5 px-2 py-1.5 text-xs text-foreground/80 outline-none focus:border-[var(--workspace-accent)] disabled:cursor-not-allowed disabled:opacity-45";

export function TextField({ label, value, onChange, multiline = false, placeholder = "", disabled = false }: { label: string; value: string; onChange: (value: string) => void; multiline?: boolean; placeholder?: string; disabled?: boolean }) {
    return (
        <label className="block">
            <span className="mb-1 block text-[var(--fs-micro)] text-foreground/42">{label}</span>
            {multiline ? (
                <textarea rows={3} className={inputClassName} value={value} placeholder={placeholder} disabled={disabled} onChange={(event) => onChange(event.target.value)} />
            ) : (
                <input className={inputClassName} value={value} placeholder={placeholder} disabled={disabled} onChange={(event) => onChange(event.target.value)} />
            )}
        </label>
    );
}

export function NumberField({ label, value, onChange, min, step = "any" }: { label: string; value: number; onChange: (value: number) => void; min?: number; step?: number | "any" }) {
    return (
        <label className="block">
            <span className="mb-1 block text-[var(--fs-micro)] text-foreground/42">{label}</span>
            <input
                className={inputClassName}
                type="number"
                value={Number.isFinite(value) ? value : 0}
                min={min}
                step={step}
                onChange={(event) => {
                    const next = Number(event.target.value);
                    onChange(Number.isFinite(next) ? next : 0);
                }}
            />
        </label>
    );
}

export function SelectField({ label, value, onChange, options }: { label: string; value: string; onChange: (value: string) => void; options: Array<{ value: string; label: string }> }) {
    const normalizedOptions = options.some((option) => option.value === value) || !value ? options : [{ value, label: `${value}（待修复）` }, ...options];
    return (
        <label className="block">
            <span className="mb-1 block text-[var(--fs-micro)] text-foreground/42">{label}</span>
            <select className={inputClassName} value={value} onChange={(event) => onChange(event.target.value)}>
                {normalizedOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                        {option.label}
                    </option>
                ))}
            </select>
        </label>
    );
}

export function TextListField({ label, values, onChange, placeholder = "每行一项" }: { label: string; values: string[]; onChange: (values: string[]) => void; placeholder?: string }) {
    return <TextField label={label} value={values.join("\n")} multiline placeholder={placeholder} onChange={(value) => onChange(linesToList(value))} />;
}

export function NumberListField({ label, values, onChange, placeholder = "每行一个数字" }: { label: string; values: number[]; onChange: (values: number[]) => void; placeholder?: string }) {
    return (
        <TextField
            label={label}
            value={values.join("\n")}
            multiline
            placeholder={placeholder}
            onChange={(value) =>
                onChange(
                    value
                        .split(/\n|、/)
                        .map((item) => Number(item.trim()))
                        .filter((item) => Number.isFinite(item)),
                )
            }
        />
    );
}

export function CheckboxField({ label, checked, onChange }: { label: string; checked: boolean; onChange: (checked: boolean) => void }) {
    return (
        <label className="flex min-h-8 cursor-pointer items-center gap-2 text-xs text-foreground/70">
            <input type="checkbox" className="size-3.5 accent-[var(--workspace-accent)]" checked={checked} onChange={(event) => onChange(event.target.checked)} />
            <span>{label}</span>
        </label>
    );
}

export function PointFields({ value, onChange, label = "坐标" }: { value: SpatialPointDraft; onChange: (value: SpatialPointDraft) => void; label?: string }) {
    const setAxis = (axis: keyof SpatialPointDraft, rawValue: string) => {
        const parsed = Number(rawValue);
        onChange({ ...value, [axis]: Number.isFinite(parsed) ? parsed : 0 });
    };
    return (
        <div>
            <span className="mb-1 block text-[var(--fs-micro)] text-foreground/42">{label}</span>
            <div className="grid grid-cols-3 gap-1.5">
                <label className="text-[var(--fs-micro)] text-foreground/38">
                    X<input aria-label={`${label} X`} className={`${inputClassName} mt-1`} type="number" value={value.x} step="any" onChange={(event) => setAxis("x", event.target.value)} />
                </label>
                <label className="text-[var(--fs-micro)] text-foreground/38">
                    Y<input aria-label={`${label} Y`} className={`${inputClassName} mt-1`} type="number" value={value.y} step="any" onChange={(event) => setAxis("y", event.target.value)} />
                </label>
                <label className="text-[var(--fs-micro)] text-foreground/38">
                    Z<input aria-label={`${label} Z`} className={`${inputClassName} mt-1`} type="number" value={value.z} step="any" onChange={(event) => setAxis("z", event.target.value)} />
                </label>
            </div>
        </div>
    );
}

export function EditorGroup({ title, detail, onAdd, addLabel = "添加", children }: { title: string; detail?: string; onAdd?: () => void; addLabel?: string; children: ReactNode }) {
    return (
        <section className="border-t border-border/70 py-3 first:border-t-0 first:pt-0">
            <div className="mb-2 flex items-start justify-between gap-2">
                <div>
                    <h4 className="text-xs font-semibold text-foreground/80">{title}</h4>
                    {detail ? <p className="mt-0.5 text-[var(--fs-micro)] leading-4 text-foreground/42">{detail}</p> : null}
                </div>
                {onAdd ? (
                    <Button size="small" icon={<Plus className="size-3.5" />} onClick={onAdd}>
                        {addLabel}
                    </Button>
                ) : null}
            </div>
            {children}
        </section>
    );
}

export function EditorItem({ title, onRemove, children }: { title: string; onRemove?: () => void; children: ReactNode }) {
    return (
        <div className="border-t border-border/60 py-3 first:border-t-0 first:pt-0">
            <div className="mb-2 flex items-center justify-between gap-2">
                <span className="truncate text-[var(--fs-caption)] font-medium text-foreground/68">{title}</span>
                {onRemove ? (
                    <Button size="small" type="text" danger icon={<Trash2 className="size-3.5" />} title="删除" aria-label={`删除${title}`} onClick={onRemove}>
                        删除
                    </Button>
                ) : null}
            </div>
            <div className="space-y-2">{children}</div>
        </div>
    );
}

export function replaceAt<T>(items: T[], index: number, value: T) {
    return items.map((item, itemIndex) => (itemIndex === index ? value : item));
}
export function removeAt<T>(items: T[], index: number) {
    return items.filter((_, itemIndex) => itemIndex !== index);
}

function linesToList(value: string) {
    return value
        .split(/\n|、/)
        .map((item) => item.trim())
        .filter(Boolean);
}
