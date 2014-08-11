module RedClothArvadosLinkExtension

  class RedClothViewBase < ActionView::Base
    include ApplicationHelper
    include ActionView::Helpers::UrlHelper
    include Rails.application.routes.url_helpers

    def helper_link_to_if_arvados_object(link, opts)
      link_to_if_arvados_object(link, opts)
    end
  end

  def refs_arvados(text)
    text.gsub!(/"(?!\s)([^"]*\S)":(\S+)/) do
      text, link = $~[1..2]
      arvados_link = RedClothViewBase.new.helper_link_to_if_arvados_object(link, { :link_text => text })
      arvados_link ? arvados_link : "#{text}:#{link}"
    end
  end
end

RedCloth.send(:include, RedClothArvadosLinkExtension)
