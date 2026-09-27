class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.69"
  license "MIT"

  # RULE: never use sha256 :no_check — Homebrew rejects it; digests must match Release assets.
  # After each release: shasum -a 256 dist/netductor-op-* and paste below (same PR as tag).
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.69/netductor-op-darwin-arm64"
      sha256 "142d9c6085fbe54a4a8b1cb3fe494596d1c084c572f93b8020ba9da51b9ee2f7"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.69/netductor-op-darwin-amd64"
      sha256 "1313997d800c2d5b92b99f2b67bb203d4be22a534f8ab5c1ecffcf19c6fdbff6"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.69/netductor-op-linux-amd64"
      sha256 "73bcc76016a2ba9e93ca1dc1874d6f125bfca5e25b04849cc217cfb3a7b7e48b"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.69/netductor-op-linux-arm64"
      sha256 "7d1faa64e2b0ac4607e77e124bf3a051bafc7f63c226178fd010a50b96926931"
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
