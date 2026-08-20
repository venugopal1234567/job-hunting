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
        // Material You / Lumina Talent Palette
        primary: {
          DEFAULT: '#4800b2',
          container: '#6200ee',
          fixed: '#e8ddff',
          'fixed-dim': '#cfbdff',
        },
        'on-primary': {
          DEFAULT: '#ffffff',
          container: '#d0beff',
          fixed: '#22005d',
          'fixed-variant': '#5300cd',
        },
        'inverse-primary': '#cfbdff',

        secondary: {
          DEFAULT: '#9e4039',
          container: '#fb877d',
          fixed: '#ffdad6',
          'fixed-dim': '#ffb4ac',
        },
        'on-secondary': {
          DEFAULT: '#ffffff',
          container: '#731f1c',
          fixed: '#410003',
          'fixed-variant': '#7f2924',
        },

        tertiary: {
          DEFAULT: '#004545',
          container: '#005e5e',
          fixed: '#93f2f2',
          'fixed-dim': '#76d6d5',
        },
        'on-tertiary': {
          DEFAULT: '#ffffff',
          container: '#78d8d7',
          fixed: '#002020',
          'fixed-variant': '#004f4f',
        },

        surface: {
          DEFAULT: '#f9f9fe',
          dim: '#d9dade',
          bright: '#f9f9fe',
          variant: '#e2e2e7',
          'container-lowest': '#ffffff',
          'container-low': '#f3f3f8',
          container: '#ededf2',
          'container-high': '#e7e8ed',
          'container-highest': '#e2e2e7',
        },
        'on-surface': {
          DEFAULT: '#1a1c1f',
          variant: '#494456',
        },
        'inverse-surface': '#2e3034',
        'inverse-on-surface': '#f0f0f5',

        background: '#f9f9fe',
        'on-background': '#1a1c1f',

        outline: {
          DEFAULT: '#7a7488',
          variant: '#cbc3d9',
        },
        'surface-tint': '#6d23f9',

        error: {
          DEFAULT: '#ba1a1a',
          container: '#ffdad6',
        },
        'on-error': {
          DEFAULT: '#ffffff',
          container: '#93000a',
        },

        // Legacy brand fallback compatibility
        brand: {
          50:  '#f5f3ff',
          100: '#ede9fe',
          200: '#ddd6fe',
          300: '#c4b5fd',
          400: '#a78bfa',
          500: '#8b5cf6',
          600: '#6d23f9',
          700: '#4800b2',
          800: '#3b008d',
          900: '#22005d',
        },
      },
      fontFamily: {
        sans: ['Google Sans', 'Inter', 'system-ui', 'sans-serif'],
        headline: ['Google Sans', 'Inter', 'sans-serif'],
        body: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      borderRadius: {
        'sm': '0.5rem',
        DEFAULT: '1rem',
        'md': '1.5rem',
        'lg': '2rem',
        'xl': '3rem',
        'full': '9999px',
        'card': '24px',
      },
      spacing: {
        'pill-padding-x': '20px',
        'pill-padding-y': '12px',
        'card-padding': '24px',
        'gutter': '24px',
        'container-margin-desktop': '32px',
        'container-margin-mobile': '16px',
        'unit': '8px',
      },
      boxShadow: {
        'elevation-1': '0 2px 4px rgba(0,0,0,0.05)',
        'elevation-2': '0 8px 16px rgba(98,0,238,0.08)',
        'elevation-3': '0 12px 24px rgba(0,0,0,0.12)',
        'soft': '0 4px 20px -2px rgba(72, 0, 178, 0.05)',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'fade-in': 'fadeIn 0.25s ease-in-out',
        'slide-up': 'slideUp 0.3s cubic-bezier(0.16, 1, 0.3, 1)',
        'spin-slow': 'spin 3s linear infinite',
        'shimmer': 'shimmer 3s linear infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '0% center' },
          '100%': { backgroundPosition: '200% center' },
        },
      },
    },
  },
  plugins: [],
}

