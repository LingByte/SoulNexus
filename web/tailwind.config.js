/** @type {import('tailwindcss').Config} */
export default {
    content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
    darkMode: "class",
    theme: {
        extend: {
            // === 现代专业色彩系统 ===
            colors: {
                // 基础色彩系统
                background: "hsl(var(--background))",
                foreground: "hsl(var(--foreground))",
                card: {
                    DEFAULT: "hsl(var(--card))",
                    foreground: "hsl(var(--card-foreground))",
                },
                popover: {
                    DEFAULT: "hsl(var(--popover))",
                    foreground: "hsl(var(--popover-foreground))",
                },
                primary: {
                    DEFAULT: "hsl(var(--primary))",
                    foreground: "hsl(var(--primary-foreground))",
                },
                secondary: {
                    DEFAULT: "hsl(var(--secondary))",
                    foreground: "hsl(var(--secondary-foreground))",
                },
                muted: {
                    DEFAULT: "hsl(var(--muted))",
                    foreground: "hsl(var(--muted-foreground))",
                },
                accent: {
                    DEFAULT: "hsl(var(--accent))",
                    foreground: "hsl(var(--accent-foreground))",
                },
                destructive: {
                    DEFAULT: "hsl(var(--destructive))",
                    foreground: "hsl(var(--destructive-foreground))",
                },
                border: "hsl(var(--border))",
                input: "hsl(var(--input))",
                ring: "hsl(var(--ring))",
            },


            // 现代圆角系统
            borderRadius: {
                'lg': 'var(--radius)',
                'md': 'calc(var(--radius) - 2px)',
                'sm': 'calc(var(--radius) - 4px)',
            },

            // 保留原有颜色系统
            neutral: {
                50: "#fafafa",
                100: "#f5f5f5",
                200: "#e5e5e5",
                300: "#d4d4d4",
                400: "#a3a3a3",
                500: "#737373",
                600: "#525252",
                700: "#404040",
                800: "#262626",
                900: "#171717",
            },
        },

        fontFamily: {
            sans: ["Inter", "system-ui", "sans-serif"],
            display: ["Poppins", "system-ui", "sans-serif"],
        },

        backgroundImage: {
            "gradient-radial": "radial-gradient(var(--tw-gradient-stops))",
            "gradient-conic": "conic-gradient(from 180deg at 50% 50%, var(--tw-gradient-stops))",
            "soft-gradient": "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
            "warm-gradient": "linear-gradient(135deg, #f093fb 0%, #f5576c 100%)",
            "cool-gradient": "linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)",
        },

        boxShadow: {
            soft: "0 2px 15px -3px rgba(0, 0, 0, 0.07), 0 10px 20px -2px rgba(0, 0, 0, 0.04)",
            "soft-lg": "0 10px 25px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)",
            glow: "0 0 20px rgba(59, 130, 246, 0.5)",
            "glow-lg": "0 0 40px rgba(59, 130, 246, 0.3)",
        },

        // === 合并后的 keyframes（保留全部） ===
        keyframes: {
            fadeIn: {
                "0%": {opacity: "0"},
                "100%": {opacity: "1"},
            },
            fadeInUp: {
                "0%": {opacity: "0", transform: "translateY(20px)"},
                "100%": {opacity: "1", transform: "translateY(0)"},
            },
            fadeInDown: {
                "0%": {opacity: "0", transform: "translateY(-20px)"},
                "100%": {opacity: "1", transform: "translateY(0)"},
            },
            slideInLeft: {
                "0%": {opacity: "0", transform: "translateX(-20px)"},
                "100%": {opacity: "1", transform: "translateX(0)"},
            },
            slideInRight: {
                "0%": {opacity: "0", transform: "translateX(20px)"},
                "100%": {opacity: "1", transform: "translateX(0)"},
            },
            scaleIn: {
                "0%": {opacity: "0", transform: "scale(0.9)"},
                "100%": {opacity: "1", transform: "scale(1)"},
            },
            bounceIn: {
                "0%": {opacity: "0", transform: "scale(0.3)"},
                "50%": {opacity: "1", transform: "scale(1.05)"},
                "70%": {transform: "scale(0.9)"},
                "100%": {opacity: "1", transform: "scale(1)"},
            },
            float: {
                "0%, 100%": {transform: "translateY(0px)"},
                "50%": {transform: "translateY(-10px)"},
            },
            gradient: {
                "0%, 100%": {backgroundPosition: "0% 50%"},
                "50%": {backgroundPosition: "100% 50%"},
            },
            // 手风琴
            "accordion-down": {
                from: {height: 0},
                to: {height: "var(--radix-accordion-content-height)"},
            },
            "accordion-up": {
                from: {height: "var(--radix-accordion-content-height)"},
                to: {height: 0},
            },
        },

        // === 合并后的 animation（保留全部） ===
        animation: {
            "fade-in": "fadeIn 0.5s ease-in-out",
            "fade-in-up": "fadeInUp 0.6s ease-out",
            "fade-in-down": "fadeInDown 0.6s ease-out",
            "slide-in-left": "slideInLeft 0.5s ease-out",
            "slide-in-right": "slideInRight 0.5s ease-out",
            "scale-in": "scaleIn 0.3s ease-out",
            "bounce-in": "bounceIn 0.6s ease-out",
            float: "float 3s ease-in-out infinite",
            "pulse-slow": "pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite",
            gradient: "gradient 15s ease infinite",
            "accordion-down": "accordion-down 0.2s ease-out",
            "accordion-up": "accordion-up 0.2s ease-out",
        },
    },
    plugins: [],
}
