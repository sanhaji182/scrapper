import type { ButtonHTMLAttributes } from "react";

export function Button({ className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return <button className={`primary-button px-4 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-60 ${className}`} {...props} />;
}
