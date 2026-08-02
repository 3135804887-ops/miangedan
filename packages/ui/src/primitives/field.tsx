import { cloneElement, type ReactElement, type ReactNode } from 'react';

import { ErrorIcon } from '../a11y/status-icons.tsx';

export interface FieldProps {
  readonly fieldId: string;
  readonly label: string;
  readonly description?: string;
  readonly errorMessage?: string;
  readonly required?: boolean;
  readonly children: ReactElement<{
    readonly id?: string;
    readonly 'aria-describedby'?: string;
    readonly 'aria-invalid'?: boolean;
  }>;
}

/** 表单字段语义容器；输入控件使用 fieldId，并关联说明与字段级错误。 */
export function Field({
  fieldId,
  label,
  description,
  errorMessage,
  required = false,
  children,
}: FieldProps): ReactNode {
  const describedBy = [
    description === undefined ? undefined : `${fieldId}-description`,
    errorMessage === undefined ? undefined : `${fieldId}-error`,
  ].filter((id): id is string => id !== undefined).join(' ');
  const control = cloneElement(children, {
    id: fieldId,
    'aria-describedby': describedBy.length > 0 ? describedBy : undefined,
    'aria-invalid': errorMessage === undefined ? undefined : true,
  });

  return (
    <div className="mgd-field" data-mgd-field={fieldId}>
      <label htmlFor={fieldId}>
        {label}
        {required ? <span aria-hidden="true"> *</span> : null}
      </label>
      {description === undefined ? null : <p id={`${fieldId}-description`}>{description}</p>}
      {control}
      {errorMessage === undefined ? null : (
        <p id={`${fieldId}-error`} role="alert" className="mgd-inline-error">
          <ErrorIcon />
          <span>{errorMessage}</span>
        </p>
      )}
    </div>
  );
}
