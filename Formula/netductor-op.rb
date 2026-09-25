class NetductorOp < Formula
  desc "netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.17"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.17/netductor-op-darwin-arm64"
      sha256 "f3cd88dc2a6a149a259994f31082258e779beb1ec4dfbf0bbbabc69576cf7512"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.17/netductor-op-darwin-amd64"
      sha256 "104a314f03b8a5bf9782af6b7ca3835f95557f63c268b5169e51bc92eb4e9470"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.17/netductor-op-linux-amd64"
      sha256 "ce3a7845e24cbf942244011293e6a393a71a741282af2cc235ee6c88731e2eda"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.17/netductor-op-linux-arm64"
      sha256 "c3f880392d299758770011600a734338037d5208364177662045f64e7c944f09"
    end
  end
  def install
    bin.install Dir["netductor-op*"].first => "netductor-op"
  end
end
