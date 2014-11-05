App.ProxyEndpointsIndexRoute = Ember.Route.extend({
  beforeModel: function() {
    this.transitionTo('newProxyEndpoint');
  }
});

App.ProxyEndpoint = DS.Model.extend({
  name: DS.attr(),
  script: DS.attr()
});

App.ProxyEndpointsRoute = Ember.Route.extend({
  model: function() {
    return this.store.find('proxyEndpoint');
  }
});

App.ProxyEndpointsController = Ember.ArrayController.extend({
  sortProperties: ['name'],
  sortAscending: true
})

App.ProxyEndpointRoute = Ember.Route.extend({
  model: function(params) {
    return this.store.find('proxyEndpoint', params.endpoint_id);
  }
})

App.ProxyEndpointController = Ember.ObjectController.extend({
  needs: ['admin'],

  actions: {
    save: function() {
      var self = this;
      this.model.save().then(function(value) {
        self.set('controllers.admin.successMessage', "Saved!");
        self.set('controllers.admin.errorMessage', null);
      }, function(reason) {
        self.set('controllers.admin.successMessage', null);
        self.set('controllers.admin.errorMessage', reason.responseText);
      });
    },
    delete: function() {
      if (confirm("Delete the endpoint '" + this.model.get('name') + "'?")) {
        this.model.destroyRecord();
        this.set('controllers.admin.successMessage', "Deleted!");
        this.set('controllers.admin.errorMessage', null);
        this.transitionToRoute('proxyEndpoints');
      }
    }
  }
});

App.NewProxyEndpointRoute = Ember.Route.extend({
  templateName: 'proxyEndpoint',
  model: function(params) {
    return this.store.createRecord('proxyEndpoint');
  }
})

App.NewProxyEndpointController = Ember.ObjectController.extend({
  // This is almost entirely duplicated from ProxyEndpointController,
  // but specifying controllerName in my route wouldn't resolve the
  // 'save' action.

  needs: ['admin'],

  actions: {
    save: function() {
      var self = this;
      this.model.save().then(function(value) {
        self.set('controllers.admin.successMessage', "Created!");
        self.set('controllers.admin.errorMessage', null);
        self.transitionToRoute("proxyEndpoint", value.id)
      }, function(reason) {
        self.set('controllers.admin.successMessage', null);
        self.set('controllers.admin.errorMessage', reason.responseText);
      });
    }
  }
});
