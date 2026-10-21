# Image Optimization Guide

## Overview

AssetForge uses Next.js Image optimization with configured remote patterns for external image sources. This guide explains how to properly use images in the application.

## Remote Image Domains

### Configured Domains

The following domains are pre-configured for remote image loading:

- `ipfs.io` - IPFS gateway
- `gateway.pinata.cloud` - Pinata IPFS gateway
- `cloudflare-ipfs.com` - Cloudflare IPFS gateway
- `images.unsplash.com` - Unsplash images

### Adding Additional Domains

To add more image domains at runtime, use the `NEXT_PUBLIC_ADDITIONAL_IMAGE_DOMAINS` environment variable:

```env
NEXT_PUBLIC_ADDITIONAL_IMAGE_DOMAINS=example.com,cdn.example.com,assets.mysite.com
```

Domains should be comma-separated without protocols or paths.

## Using the OptimizedImage Component

### Basic Usage

```tsx
import { OptimizedImage } from "@/components/OptimizedImage";

export function MyComponent() {
  return (
    <OptimizedImage
      src="https://ipfs.io/ipfs/QmHash..."
      alt="Asset image"
      width={500}
      height={300}
    />
  );
}
```

### With Fallback

```tsx
<OptimizedImage
  src="https://gateway.pinata.cloud/ipfs/QmHash..."
  alt="NFT artwork"
  width={800}
  height={600}
  fallbackSrc="/images/placeholder.png"
  onError={() => console.log("Image failed to load")}
/>
```

### Responsive Images

```tsx
<OptimizedImage
  src="https://images.unsplash.com/photo-123"
  alt="Hero image"
  fill
  sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"
  className="object-cover"
/>
```

## Image Formats

Next.js automatically serves images in the most efficient format:

1. **AVIF** - Modern format with best compression (when browser supports)
2. **WebP** - Fallback format with good compression
3. **Original format** - Used when modern formats aren't supported

## Best Practices

### 1. Always Specify Dimensions

```tsx
// ✅ Good - prevents layout shift
<OptimizedImage src="..." alt="..." width={400} height={300} />

// ❌ Bad - causes layout shift
<OptimizedImage src="..." alt="..." />
```

### 2. Use Appropriate Sizes Prop

```tsx
// For responsive images, specify sizes
<OptimizedImage
  src="..."
  alt="..."
  fill
  sizes="(max-width: 768px) 100vw, 50vw"
/>
```

### 3. Provide Meaningful Alt Text

```tsx
// ✅ Good - descriptive
<OptimizedImage src="..." alt="Rare painting by Vincent van Gogh" />

// ❌ Bad - not descriptive
<OptimizedImage src="..." alt="image" />
```

### 4. Use Placeholders for Loading States

```tsx
<OptimizedImage
  src="..."
  alt="..."
  width={500}
  height={300}
  placeholder="blur"
  blurDataURL="data:image/png;base64,..."
/>
```

### 5. Handle Errors Gracefully

```tsx
<OptimizedImage
  src={userProvidedUrl}
  alt="User upload"
  width={300}
  height={300}
  fallbackSrc="/images/default-avatar.png"
  onError={() => {
    // Log error to monitoring service
    console.error("Image load failed");
  }}
/>
```

## CORS and Security

### Remote Pattern Format

Next.js 16 requires explicit remote patterns with all fields:

```typescript
{
  protocol: "https",
  hostname: "example.com",
  port: "",        // Required (empty string for default)
  pathname: "/**", // Required (/** allows all paths)
}
```

### Security Considerations

1. **Only allow trusted domains** - Don't add arbitrary user-provided domains
2. **Use HTTPS only** - Never allow HTTP image sources in production
3. **Validate image sources** - Check domain allowlist before rendering user content
4. **Content Security Policy** - Ensure CSP headers allow configured image domains

## Troubleshooting

### Images Not Loading

1. **Check remote pattern configuration** in `frontend/next.config.ts`
2. **Verify domain is in allowlist** or `NEXT_PUBLIC_ADDITIONAL_IMAGE_DOMAINS`
3. **Check browser console** for CORS errors
4. **Ensure image URL is valid** and returns 200 status

### CORS Errors

If you see CORS errors:

```
Access to image at 'https://example.com/image.jpg' from origin 'http://localhost:3000' 
has been blocked by CORS policy
```

This means:
- The image server doesn't allow cross-origin requests
- You need to proxy the image through your backend
- Or use a different image source

### Domain Not Allowed Error

```
Error: Invalid src prop (https://newdomain.com/image.jpg) on `next/image`, 
hostname "newdomain.com" is not configured under images in your `next.config.js`
```

Solution:
1. Add domain to `remotePatterns` in `next.config.ts`, or
2. Add to `NEXT_PUBLIC_ADDITIONAL_IMAGE_DOMAINS` environment variable

## Performance Tips

1. **Use appropriate image sizes** - Don't load 4K images for thumbnails
2. **Lazy load below-the-fold images** - Use `loading="lazy"` prop
3. **Prioritize hero images** - Use `priority` prop for above-the-fold content
4. **Consider image CDN** - For user-uploaded content, use a CDN with resizing

## Example: Asset Card with Image

```tsx
import { OptimizedImage } from "@/components/OptimizedImage";
import { Card } from "@/components/ui/card";

interface AssetCardProps {
  title: string;
  imageUrl: string;
  price: string;
}

export function AssetCard({ title, imageUrl, price }: AssetCardProps) {
  return (
    <Card className="overflow-hidden">
      <div className="relative aspect-square">
        <OptimizedImage
          src={imageUrl}
          alt={title}
          fill
          sizes="(max-width: 768px) 100vw, (max-width: 1200px) 50vw, 33vw"
          className="object-cover"
          fallbackSrc="/images/asset-placeholder.png"
        />
      </div>
      <div className="p-4">
        <h3 className="font-semibold">{title}</h3>
        <p className="text-muted-foreground">{price}</p>
      </div>
    </Card>
  );
}
```

## References

- [Next.js Image Optimization](https://nextjs.org/docs/app/building-your-application/optimizing/images)
- [Next.js Image Configuration](https://nextjs.org/docs/app/api-reference/components/image)
- [Remote Patterns Configuration](https://nextjs.org/docs/app/api-reference/next-config-js/images#remotepatterns)
