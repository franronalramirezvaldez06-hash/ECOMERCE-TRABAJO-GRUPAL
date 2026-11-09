import { forwardRef, useId, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from "react";
import clsx from "clsx";

const CONTROL =
  "w-full rounded-lg border bg-white px-3 text-sm text-slate-900 placeholder:text-slate-400 disabled:bg-slate-100 disabled:text-slate-500";

function controlClasses(error?: string, extra?: string) {
  return clsx(CONTROL, error ? "border-red-400" : "border-slate-300", extra);
}

interface FieldProps {
  label: string;
  error?: string;
  hint?: string;
  children: (props: { id: string; "aria-invalid": boolean; "aria-describedby": string | undefined }) => ReactNode;
}

/** Envuelve un control con etiqueta, ayuda y error accesibles. */
export function Field({ label, error, hint, children }: FieldProps) {
  const id = useId();
  const describedBy = error || hint ? `${id}-desc` : undefined;
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      {children({ id, "aria-invalid": !!error, "aria-describedby": describedBy })}
      {(error || hint) && (
        <p id={describedBy} className={clsx("text-xs", error ? "text-red-600" : "text-slate-500")}>
          {error ?? hint}
        </p>
      )}
    </div>
  );
}

interface TextFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  hint?: string;
}

export const TextField = forwardRef<HTMLInputElement, TextFieldProps>(function TextField(
  { label, error, hint, className, ...props },
  ref,
) {
  return (
    <Field label={label} error={error} hint={hint}>
      {(a11y) => <input ref={ref} className={controlClasses(error, clsx("h-10", className))} {...a11y} {...props} />}
    </Field>
  );
});

interface TextAreaFieldProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label: string;
  error?: string;
  hint?: string;
}

export function TextAreaField({ label, error, hint, className, ...props }: TextAreaFieldProps) {
  return (
    <Field label={label} error={error} hint={hint}>
      {(a11y) => <textarea rows={3} className={controlClasses(error, clsx("py-2", className))} {...a11y} {...props} />}
    </Field>
  );
}

interface SelectFieldProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label: string;
  error?: string;
  hint?: string;
}

export function SelectField({ label, error, hint, className, children, ...props }: SelectFieldProps) {
  return (
    <Field label={label} error={error} hint={hint}>
      {(a11y) => (
        <select className={controlClasses(error, clsx("h-10", className))} {...a11y} {...props}>
          {children}
        </select>
      )}
    </Field>
  );
}
