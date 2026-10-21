import type { Metadata } from 'next';
import type { ReactNode } from 'react';

// The title is rendered through the root layout's "%s | kor-AssetForge" template.
export const metadata: Metadata = {
  title: 'Admin Panel',
  description: 'Platform management and monitoring dashboard',
  robots: {
    index: false,
    follow: false,
  },
};

export default function AdminLayout({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <div className="bg-gray-50">
      {children}
    </div>
  );
}
