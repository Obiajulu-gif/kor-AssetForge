"use client"

import { SectionError } from "@/components/errors/SectionError"
import type { RouteErrorProps } from "@/components/errors/ErrorFallback"

export default function AdminError(props: RouteErrorProps) {
  return <SectionError {...props} section="Admin" />
}
