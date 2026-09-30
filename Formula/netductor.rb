class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.122"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.122/netductor-op-darwin-arm64"
      sha256 "4c59ea06e252e9a5e978ec3fe848d0578ee886dc8e46165eb42bf36fa3b89581"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.122/netductor-op-darwin-amd64"
      sha256 "8cad59f416a435a0f3404d4cc85d91ab4497aa16506a8e2fb6f77c61671b9924"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.122/netductor-op-linux-amd64"
      sha256 "f1184f7fbe910dbd2075388bd5cfcbf2e7a6213349829ca322c6ec7d4374086a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
