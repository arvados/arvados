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
      # if it's not an arvados_link the helper will return the link unprocessed and so we will reconstruct the textile link string so it can be processed normally
      (arvados_link == link) ? "\"#{text}\":#{link}" : arvados_link
    end
  end
end

RedCloth.send(:include, RedClothArvadosLinkExtension)
