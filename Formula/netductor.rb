class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.247"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.247/netductor-op-darwin-arm64"
      sha256 "162dce7c19a5a1c195eb2015a4ec738407fffe7743b25bff94a563805d8eed78"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.247/netductor-op-darwin-amd64"
      sha256 "398167329dada90cd455c6645cb9d05b5196c3548720e31c4704581048e2ef98"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.247/netductor-op-linux-amd64"
      sha256 "a2b622a162d9e3888fd9a6fcae9f352df486b32f2506a9c1a63d71cac9eb888d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
