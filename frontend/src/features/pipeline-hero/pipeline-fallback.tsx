import { cn } from "@/lib/utils/cn";

export function PipelineFallback({ className }: { className?: string }) {
  return (
    <svg
      aria-hidden="true"
      className={cn("size-full", className)}
      fill="none"
      viewBox="0 0 800 560"
    >
      <defs>
        <linearGradient
          id="pipeline-silver"
          x1="180"
          x2="640"
          y1="130"
          y2="430"
        >
          <stop stopColor="#73757C" />
          <stop offset=".42" stopColor="#D5D6D9" />
          <stop offset=".72" stopColor="#92949A" />
          <stop offset="1" stopColor="#E4E4E5" />
        </linearGradient>
        <linearGradient id="pipeline-dark" x1="100" x2="700" y1="420" y2="120">
          <stop stopColor="#08090A" />
          <stop offset=".55" stopColor="#28292E" />
          <stop offset="1" stopColor="#08090A" />
        </linearGradient>
        <radialGradient id="pipeline-glow">
          <stop stopColor="#50525A" stopOpacity=".48" />
          <stop offset="1" stopColor="#050505" stopOpacity="0" />
        </radialGradient>
        <filter
          id="pipeline-shadow"
          x="-30%"
          y="-30%"
          width="160%"
          height="170%"
        >
          <feDropShadow
            dx="0"
            dy="22"
            floodColor="#000"
            floodOpacity=".6"
            stdDeviation="18"
          />
        </filter>
        <filter id="node-glow" x="-300%" y="-300%" width="700%" height="700%">
          <feGaussianBlur stdDeviation="7" />
        </filter>
      </defs>

      <ellipse cx="420" cy="294" fill="url(#pipeline-glow)" rx="370" ry="250" />
      <g filter="url(#pipeline-shadow)" strokeLinecap="round">
        <path
          d="M78 373C206 285 304 315 405 270C510 224 618 170 742 101"
          stroke="#060708"
          strokeWidth="48"
        />
        <path
          d="M70 365C205 281 307 309 407 264C516 216 617 167 742 96"
          stroke="url(#pipeline-dark)"
          strokeWidth="37"
        />
        <path
          d="M112 174C164 72 244 92 324 111C417 133 487 106 557 71C610 45 668 34 741 25"
          stroke="url(#pipeline-silver)"
          strokeWidth="42"
        />
        <path
          d="M91 485C229 374 345 395 451 393C558 391 649 425 730 471"
          stroke="#050607"
          strokeWidth="49"
        />
        <path
          d="M89 478C229 369 347 389 453 387C559 385 649 419 730 464"
          stroke="url(#pipeline-dark)"
          strokeWidth="37"
        />
        <path
          d="M167 453C278 391 363 325 444 268C520 215 597 169 688 122"
          stroke="#6F7178"
          strokeWidth="46"
        />
        <path
          d="M169 447C280 385 364 319 446 262C522 209 599 163 690 116"
          stroke="url(#pipeline-silver)"
          strokeWidth="34"
        />
        <path
          d="M79 286C197 228 309 211 409 231C523 254 623 235 738 175"
          stroke="#070809"
          strokeWidth="39"
        />
        <path
          d="M79 280C198 222 310 205 410 225C524 248 623 229 738 169"
          stroke="url(#pipeline-dark)"
          strokeWidth="29"
        />
      </g>

      <g fill="none" stroke="#9B725E" strokeWidth="12">
        <path d="M275 385L297 401" />
        <path d="M444 262L465 279" />
        <path d="M586 174L607 191" />
        <path d="M133 127L151 139" />
        <path d="M552 74L564 94" />
      </g>

      <g fill="#8DA8E8">
        <circle
          cx="248"
          cy="248"
          opacity=".4"
          r="15"
          filter="url(#node-glow)"
        />
        <circle cx="248" cy="248" r="6" />
        <circle
          cx="561"
          cy="212"
          opacity=".4"
          r="15"
          filter="url(#node-glow)"
        />
        <circle cx="561" cy="212" r="6" />
        <circle
          cx="647"
          cy="407"
          opacity=".4"
          r="15"
          filter="url(#node-glow)"
        />
        <circle cx="647" cy="407" r="6" />
      </g>
    </svg>
  );
}
