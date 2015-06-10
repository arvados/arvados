package org.arvados.sdk;

import com.google.api.client.util.Lists;
import com.google.api.client.util.Sets;

import java.util.ArrayList;
import java.util.SortedSet;

public class MethodDetails implements Comparable<MethodDetails> {
    String name;
    ArrayList<String> requiredParameters = Lists.newArrayList();
    SortedSet<String> optionalParameters = Sets.newTreeSet();
    boolean hasContent;

    @Override
    public int compareTo(MethodDetails o) {
      if (o == this) {
        return 0;
      }
      return name.compareTo(o.name);
    }
}
