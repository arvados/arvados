module ApplicationHelper
  def current_user
    controller.current_user
  end
  def human_readable_bytes_html(n)
    return h(n) unless n.is_a? Fixnum
    raw = n.to_s
    cooked = ''
    while raw.length > 3
      cooked = ',' + raw[-3..-1] + cooked
      raw = raw[0..-4]
    end
    cooked = raw + cooked
  end
end
