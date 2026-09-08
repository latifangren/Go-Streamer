/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        neoCanvas: "#daf0fc",
        neoMint: "#c8f5d0",
        neoBlue: "#c2e7ff",
        neoLavender: "#e2daf9",
        neoCoral: "#ffd5cc",
        neoYellow: "#fff0a3",
        neoPink: "#ffd4e5",
        neoDark: "#111827",
      },
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'monospace'],
      },
      boxShadow: {
        'neo-sm': '2px 2px 0px #000000',
        'neo': '3px 3px 0px #000000',
        'neo-md': '4px 4px 0px #000000',
        'neo-lg': '6px 6px 0px #000000',
        'neo-xl': '8px 8px 0px #000000',
      },
      borderWidth: {
        '2.5': '2.5px',
        '3': '3px',
      },
    },
  },
  plugins: [],
}
