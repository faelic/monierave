export function AuthenticationAtmosphere() {
  return (
    <svg
      aria-hidden="true"
      className="absolute inset-0 size-full"
      fill="none"
      preserveAspectRatio="xMidYMid slice"
      viewBox="0 0 1600 1000"
    >
      <defs>
        <linearGradient
          id="auth-top-fold"
          x1="1110"
          x2="1540"
          y1="-20"
          y2="650"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#f0f0ee" stopOpacity="0.9" />
          <stop offset="0.28" stopColor="#a9aaad" stopOpacity="0.78" />
          <stop offset="0.7" stopColor="#505156" stopOpacity="0.56" />
          <stop offset="1" stopColor="#1c1d20" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="auth-top-shadow"
          x1="990"
          x2="1540"
          y1="20"
          y2="530"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#08090a" />
          <stop offset="0.48" stopColor="#4f5054" stopOpacity="0.48" />
          <stop offset="1" stopColor="#050608" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="auth-bottom-fold"
          x1="-20"
          x2="560"
          y1="620"
          y2="1060"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#ececea" stopOpacity="0.9" />
          <stop offset="0.32" stopColor="#96979b" stopOpacity="0.76" />
          <stop offset="0.75" stopColor="#37383c" stopOpacity="0.46" />
          <stop offset="1" stopColor="#111215" stopOpacity="0" />
        </linearGradient>
        <radialGradient id="auth-center-glow" cx="0" cy="0" r="1">
          <stop stopColor="#ffffff" stopOpacity="0.055" />
          <stop offset="1" stopColor="#ffffff" stopOpacity="0" />
        </radialGradient>
        <filter id="auth-softness" x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="3" />
        </filter>
        <filter id="auth-grain" x="-20%" y="-20%" width="140%" height="140%">
          <feTurbulence
            baseFrequency="0.72"
            numOctaves="3"
            seed="17"
            stitchTiles="stitch"
            type="fractalNoise"
          />
          <feColorMatrix values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 .16 0" />
          <feBlend in="SourceGraphic" mode="soft-light" />
        </filter>
      </defs>

      <ellipse
        cx="790"
        cy="500"
        fill="url(#auth-center-glow)"
        rx="610"
        ry="440"
      />

      <g filter="url(#auth-softness)" opacity="0.92">
        <path
          d="M1000 -115C1071 83 1148 221 1276 329C1393 428 1504 495 1669 548L1708 -127L1000 -115Z"
          fill="url(#auth-top-fold)"
        />
        <path
          d="M1050 -88C1133 155 1260 302 1439 426C1519 482 1592 522 1675 551L1692 633C1484 558 1324 462 1209 340C1107 232 1046 92 1010 -72L1050 -88Z"
          fill="url(#auth-top-shadow)"
          opacity="0.7"
        />
        <path
          d="M-106 506C59 659 187 760 315 862C411 939 482 1006 568 1116L-130 1104L-106 506Z"
          fill="url(#auth-bottom-fold)"
        />
        <path
          d="M-77 566C93 711 213 808 329 901C405 962 469 1027 528 1104L430 1107C351 1012 281 953 190 888C70 802 -18 724 -105 646L-77 566Z"
          fill="#0c0d0f"
          opacity="0.64"
        />
      </g>

      <g filter="url(#auth-grain)" opacity="0.27">
        <path
          d="M1000 -115C1071 83 1148 221 1276 329C1393 428 1504 495 1669 548L1708 -127L1000 -115Z"
          fill="url(#auth-top-fold)"
        />
        <path
          d="M-106 506C59 659 187 760 315 862C411 939 482 1006 568 1116L-130 1104L-106 506Z"
          fill="url(#auth-bottom-fold)"
        />
      </g>
    </svg>
  );
}
