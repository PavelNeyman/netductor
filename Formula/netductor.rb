class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.92"
  license "MIT"

  # RULE: never use sha256 :no_check — Homebrew rejects it; digests must match Release assets.
  # After each release: shasum -a 256 dist/netductor-op-* and paste below (same PR as tag).
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.92/netductor-op-darwin-arm64"
      sha256 "495134757982056185bcb83372add528b04381a0157f7d4f18a636f8785812e6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.92/netductor-op-darwin-amd64"
      sha256 "cce924a98c06893cb3e78fa28d5a734df39ad97bd4c55359f51bf99d37d41b4a"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.92/netductor-op-linux-amd64"
      sha256 "eeb56f7702a158604a2acb0902672237166a07df5ac8d1814622ff49ee2ab961"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.92/netductor-op-linux-arm64"
      sha256 "0ad7c7b41832c82b35ee55be33eff6edbc88cb20cfbb5f8f37f0cbd6555b1546"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
