/** @type {import('next').NextConfig} */
const nextConfig = {
  experimental: {
    appDir: true,
  },
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/v1/:path*`,
      },
    ]
  },
  images: {
    domains: ['localhost'],
  },
}

module.exports = nextConfig