/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/web/templates/**/*.html"],
  theme: {
    extend: {
      colors: {
        accent: {
          50:  '#fff8ed',
          100: '#ffeed4',
          200: '#ffd9a8',
          300: '#ffbe71',
          400: '#ff9838',
          500: '#ff7a11',
          600: '#f05e06',
          700: '#c74507',
          800: '#9e370e',
          900: '#7f300f',
          950: '#451605',
        },
        surface: {
          50:  '#f6f6f9',
          100: '#ededf1',
          200: '#d7d7e0',
          300: '#b4b4c5',
          400: '#8b8ba5',
          500: '#6d6d8a',
          600: '#575772',
          700: '#47475d',
          800: '#2a2a3d',
          850: '#222233',
          900: '#1a1a2e',
          950: '#12121f',
        },
      },
      fontFamily: {
        display: ['"Outfit"', 'system-ui', 'sans-serif'],
        body:    ['"DM Sans"', 'system-ui', 'sans-serif'],
      },
      animation: {
        'glow-pulse': 'glow-pulse 2s ease-in-out infinite',
        'fade-in': 'fade-in 0.3s ease-out',
        'slide-up': 'slide-up 0.4s ease-out',
      },
      keyframes: {
        'glow-pulse': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.5' },
        },
        'fade-in': {
          from: { opacity: '0' },
          to: { opacity: '1' },
        },
        'slide-up': {
          from: { opacity: '0', transform: 'translateY(8px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
      },
    },
  },
  plugins: [],
}
