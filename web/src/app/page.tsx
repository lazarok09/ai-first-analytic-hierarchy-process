import { HomeExperience } from "@/components/HomeExperience";
import { SITE } from "@/lib/site";

export default function HomePage() {
  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: SITE.name,
    applicationCategory: "DeveloperApplication",
    operatingSystem: "Linux, macOS, Windows",
    description: SITE.description,
    url: SITE.url,
    downloadUrl: SITE.releases,
    softwareVersion: "latest",
    license: SITE.license,
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "USD",
    },
    codeRepository: SITE.github,
  };

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <h1 className="sr-only">
        {SITE.name} — {SITE.tagline}
      </h1>
      <HomeExperience />
    </>
  );
}
