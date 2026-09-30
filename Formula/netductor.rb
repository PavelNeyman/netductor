class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.117"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.117/netductor-op-darwin-arm64"
      sha256 "60dd9d1b511ade75c275a6a718c89a8dc862380bb4bc2628a4df611989993807"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.117/netductor-op-darwin-amd64"
      sha256 "3aa867516024a216476d38cc8538d27c84469beff754cb437192bf6bb2c6efb8"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.117/netductor-op-linux-amd64"
      sha256 "b48ed4a25640ceb4f0307d8b8af8d960be52f43bb47503ae0f8048a4e46f896a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
