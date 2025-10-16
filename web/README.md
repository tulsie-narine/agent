# Inventory Web Console

The web console for the Windows Inventory Agent & Cloud Console system, built with Next.js 14+ and TypeScript.

## Features

- **Dashboard**: Real-time overview of device inventory and health
- **Device Management**: View, filter, and manage registered devices
- **Telemetry Visualization**: Charts and graphs for system metrics
- **Policy Management**: Create and distribute policies to devices
- **Command Execution**: Send commands to devices and monitor execution
- **User Authentication**: Secure access with role-based permissions

## Tech Stack

- **Framework**: Next.js 14+ with App Router
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **Charts**: Recharts
- **HTTP Client**: Axios
- **UI Components**: Headless UI + Heroicons

## Getting Started

1. Install dependencies:
   ```bash
   npm install
   ```

2. Copy environment variables:
   ```bash
   cp .env.example .env.local
   ```

3. Configure environment variables in `.env.local`:
   ```
   NEXT_PUBLIC_API_URL=http://localhost:8080
   NEXTAUTH_URL=http://localhost:3000
   NEXTAUTH_SECRET=your-secret-key
   ```

4. Start the development server:
   ```bash
   npm run dev
   ```

5. Open [http://localhost:3000](http://localhost:3000) in your browser.

## Project Structure

```
src/
├── app/                    # Next.js App Router pages
│   ├── (auth)/            # Authentication pages
│   ├── (dashboard)/       # Dashboard pages
│   └── api/               # API routes
├── components/            # Reusable UI components
│   ├── ui/               # Base UI components
│   ├── layout/           # Layout components
│   └── charts/           # Chart components
├── lib/                   # Utility functions and configurations
│   ├── api/              # API client functions
│   ├── auth/             # Authentication utilities
│   └── utils/            # General utilities
└── types/                 # TypeScript type definitions
```

## Development

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run start` - Start production server
- `npm run lint` - Run ESLint
- `npm run type-check` - Run TypeScript type checking

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_API_URL` | API server URL | `http://localhost:8080` |
| `NEXTAUTH_URL` | NextAuth.js URL | `http://localhost:3000` |
| `NEXTAUTH_SECRET` | NextAuth.js secret | Required |

## Contributing

1. Follow the existing code style and patterns
2. Add TypeScript types for new data structures
3. Write tests for new functionality
4. Update documentation as needed