"use client";

import { useTranslations } from "next-intl";

const SAATY_URL = "https://doi.org/10.1016/0022-2496(77)90033-5";
const GAUSSIAN_URL = "https://doi.org/10.13033/ijahp.v13i1.833";

export function ReferencesFooter() {
  const t = useTranslations("footer");

  return (
    <footer className="site-footer">
      <p className="site-footer-label">{t("references")}</p>
      <ul className="site-footer-links">
        <li>
          <a href={SAATY_URL} target="_blank" rel="noopener noreferrer">
            {t("saaty")}
          </a>
        </li>
        <li>
          <a href={GAUSSIAN_URL} target="_blank" rel="noopener noreferrer">
            {t("gaussian")}
          </a>
        </li>
      </ul>
    </footer>
  );
}
