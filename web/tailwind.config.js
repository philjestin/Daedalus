/**
 * Tailwind CSS v4 compiles theme tokens from src/index.css (@theme)
 * and src/theme.css. This file is not loaded by the Vite build.
 * @type {import('tailwindcss').Config}
 */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Custom color palette - industrial/maker aesthetic
        surface: {
          50: '#FFFFFF',
          100: '#1A1C1D',
          200: '#1A1C1D',
          300: '#2C2D2E',
          400: '#2C2D2E',
          500: '#5E6368',
          600: '#6B7076',
          700: '#E4E4E4',
          800: '#EEEEEE',
          900: '#F7F7F7',
          950: '#FFFFFF',
        },
        accent: {
          DEFAULT: '#3295FF',
          50: '#E8F3FF',
          100: '#D6EAFF',
          200: '#ADDAFF',
          300: '#7CC4FF',
          400: '#3295FF',
          500: '#3295FF',
          600: '#3295FF',
          700: '#1868B3',
          800: '#14528C',
          900: '#0E3B66',
        },
      },
      fontFamily: {
        sans: ['Roboto', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif'],
        display: ['Poppins', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif'],
      },
    },
  },
  plugins: [],
}

