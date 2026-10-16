export function Logo({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 32 32"
      fill="none"
      aria-hidden
      className={className}
    >
      <path
        d="M16 9.5v5.2M16 14.7 8.8 23.2M16 14.7l7.2 8.5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        className="text-steel"
      />
      <circle cx="16" cy="7.2" r="3.15" className="fill-amber" />
      <circle cx="8.2" cy="24.8" r="2.55" className="fill-steel-bright" />
      <circle cx="23.8" cy="24.8" r="2.55" className="fill-steel-bright" />
    </svg>
  );
}
