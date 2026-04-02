import { useState } from "react";

interface BlurredImageProps {
  src: string;
  alt?: string;
  isNsfw?: boolean;
  className?: string;
}

export function BlurredImage({ src, alt = "", isNsfw = false, className }: BlurredImageProps) {
  const [revealed, setRevealed] = useState(false);

  if (!isNsfw || revealed) {
    return <img src={src} alt={alt} className={className} />;
  }

  return (
    <div className="relative cursor-pointer" onClick={() => setRevealed(true)}>
      <img src={src} alt={alt} className={`blur-xl ${className ?? ""}`} />
      <div className="absolute inset-0 flex items-center justify-center bg-black/50 text-white text-sm">
        Click to view
      </div>
    </div>
  );
}
