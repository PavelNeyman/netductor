class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.63"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.63/netductor-darwin-arm64"
      sha256 "594014e7e3f515cfbff84e3710b339428ae1e78017c28b63d5b262cb78997d62"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.63/netductor-darwin-amd64"
      sha256 "7aca6e731647df0c18e05aad25f937d0cc3a4b5f5d10444faf557f6fddb700aa"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.63/netductor-linux-amd64"
      sha256 "0f30b9c53da32eced0e4aef810557077b9b283b444c80a1bdca297239f68cb9d"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.63/netductor-linux-arm64"
      sha256 "e00520493b9bda5e9367e97de596e8f4ae5ddd43e512810d37d4e9dface80799"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
