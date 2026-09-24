class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.87"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.87/netductor-darwin-arm64"
      sha256 "83fe8f3a7ce59634c7af750ee258996d59104398927baa31771f1bf9d5b4aad1"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.87/netductor-darwin-amd64"
      sha256 "5a07b3d201bdf57ae8fc8457a382f82babe85f0295b373af9af4b9ef3346d5b5"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.87/netductor-linux-amd64"
      sha256 "9d7b35b69a43ac397857f37ec0c126ff7f1563f188754f37ec066c6ef7945e8b"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.87/netductor-linux-arm64"
      sha256 "12be55b2cbfb67a09995ac6bdb634cf56271d4259001f3a131210d29d2f3f548"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
