"use client";

import Image, { ImageProps } from "next/image";
import { useState } from "react";

interface OptimizedImageProps extends Omit<ImageProps, "onError"> {
  fallbackSrc?: string;
  onError?: () => void;
}

/**
 * OptimizedImage - Wrapper around Next.js Image component with error handling
 * 
 * Provides automatic fallback handling for remote images that fail to load.
 * Supports all standard Next.js Image props plus custom fallback behavior.
 * 
 * @example
 * ```tsx
 * <OptimizedImage
 *   src="https://example.com/image.jpg"
 *   alt="Description"
 *   width={500}
 *   height={300}
 *   fallbackSrc="/placeholder.png"
 * />
 * ```
 */
export function OptimizedImage({
  src,
  fallbackSrc = "/placeholder.png",
  onError,
  ...props
}: OptimizedImageProps) {
  const [imgSrc, setImgSrc] = useState<string | typeof src>(src);
  const [hasError, setHasError] = useState(false);

  const handleError = () => {
    if (!hasError) {
      setHasError(true);
      setImgSrc(fallbackSrc);
      onError?.();
    }
  };

  return (
    <Image
      {...props}
      src={imgSrc}
      onError={handleError}
    />
  );
}
