# v0.9+: Mac should install netductor-op-* as netductor-op; update URLs/sha after first v0.9 release.
class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.98"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.98/netductor-darwin-arm64"
      sha256 "b70c00584749b0b56abba465536cf7b6f3f2e246b4b97152c6ea9d4f715887c2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.98/netductor-darwin-amd64"
      sha256 "dbcf84f11ab0b90b3695a4474f6ede9c7be1d9b6b40d11f62b6ba57b64dc9bbe"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.98/netductor-linux-amd64"
      sha256 "6eea84ef6b89f110a39bcb918c9393744c65360cb872f1011b412f48f696b031"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.98/netductor-linux-arm64"
      sha256 "2d52dd61f3a2948ca99feb88095ebf18386409c9a0ea148379e06ee54e9a29e8"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
