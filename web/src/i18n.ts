import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import tr from './locales/tr.json';
import en from './locales/en.json';

const STORAGE_KEY = 'tourna_desk_lang';

const getInitialLanguage = (): 'tr' | 'en' => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'en' || saved === 'tr') {
      return saved;
    }
  } catch {
    // ignore storage errors
  }
  return 'tr';
};

i18n.use(initReactI18next).init({
  resources: {
    tr: { translation: tr },
    en: { translation: en },
  },
  lng: getInitialLanguage(),
  fallbackLng: 'tr',
  interpolation: {
    escapeValue: false,
  },
});

export const setAppLanguage = (lang: 'tr' | 'en') => {
  try {
    localStorage.setItem(STORAGE_KEY, lang);
  } catch {
    // ignore storage errors
  }
  i18n.changeLanguage(lang);
};

export default i18n;
