"use client";

import React, { useTransition } from 'react';
import { useLocale } from 'next-intl';
import { setLocale } from '@/actions/locale';

const LanguageSelector: React.FC = () => {
  const locale = useLocale();
  const [isPending, startTransition] = useTransition();

  const languages = [
    { code: 'en', name: 'English' },
    { code: 'es', name: 'Español' },
    { code: 'fr', name: 'Français' },
    { code: 'zh', name: '中文' },
  ];

  const handleLanguageChange = (languageCode: string) => {
    startTransition(async () => {
      await setLocale(languageCode);
    });
  };

  return (
    <div className="language-selector">
      <select
        value={locale}
        disabled={isPending}
        onChange={(e) => handleLanguageChange(e.target.value)}
        className="language-select border rounded p-1 text-sm bg-background text-foreground"
      >
        {languages.map((lang) => (
          <option key={lang.code} value={lang.code}>
            {lang.name}
          </option>
        ))}
      </select>
    </div>
  );
};

export default LanguageSelector;
