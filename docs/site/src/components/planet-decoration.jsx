export default function PlanetDecoration() {
  return (
    <div className="planet-decoration" aria-hidden="true">
      <svg
        viewBox="0 0 200 200"
        xmlns="http://www.w3.org/2000/svg"
        className="planet-decoration__svg"
      >
        {/* planet body - drawn first so flag sits on top */}
        <circle
          cx="96"
          cy="124"
          r="72"
          fill="#ffffff"
          stroke="#111"
          strokeWidth="2.5"
        />

        {/* simple latitude lines for a globe feel, ASCII/line-art style */}
        <ellipse
          cx="96"
          cy="124"
          rx="72"
          ry="18"
          fill="none"
          stroke="#111"
          strokeWidth="1"
          opacity="0.5"
        />
        <ellipse
          cx="96"
          cy="124"
          rx="72"
          ry="36"
          fill="none"
          stroke="#111"
          strokeWidth="1"
          opacity="0.5"
        />
        <ellipse
          cx="96"
          cy="124"
          rx="72"
          ry="54"
          fill="none"
          stroke="#111"
          strokeWidth="1"
          opacity="0.5"
        />

        {/* flag pole - base on circle border (cx=96, cy=124, r=72, ~upper-right) */}
        <line
          x1="118"
          y1="55"
          x2="118"
          y2="18"
          stroke="#333"
          strokeWidth="3"
          strokeLinecap="round"
        />
        {/* flag */}
        <path
          d="M118 18 L158 28 L118 38 Z"
          fill="#000080"
          stroke="#000"
          strokeWidth="1.5"
        />
        <path
          d="M118 28 L148 36 L118 44 Z"
          fill="#c45c5c"
          stroke="#000"
          strokeWidth="1"
        />
      

       
      </svg>
    </div>
  );
}