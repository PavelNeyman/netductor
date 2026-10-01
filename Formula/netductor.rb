class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.161"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.161/netductor-op-darwin-arm64"
      sha256 "074ae35eeaf843983b2785bcd39b8183d8fafcd9c3dae0c97ef7de39b9b4a54b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.161/netductor-op-darwin-amd64"
      sha256 "0fd056f1812fcb5b5b4fedaa345c0baa80660f878864cfcccedee00afd92e0d7"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor-op version 2>&1")
  end
end
