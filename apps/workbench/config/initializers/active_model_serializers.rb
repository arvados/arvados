# Stop active_model_serializers from adding root elements to JSON responses.
ActiveModel::Serializer.root = false
ActiveModel::ArraySerializer.root = false
