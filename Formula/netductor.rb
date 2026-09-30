class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.121"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.121/netductor-op-darwin-arm64"
      sha256 "85adadd376be1804a56e78e74db38124d1d1f0a6f3cfda619580c2254ea5448f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.121/netductor-op-darwin-amd64"
      sha256 "e2dc2ed9a4d1a9bb0fd8048f6e4963c3ff714c65331eb655a25375fa0e85403c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.121/netductor-op-linux-amd64"
      sha256 "7b8490093da17d6b5d0bceceb55347d6c5841c6f927a97e3807c2c12205e0b47"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
