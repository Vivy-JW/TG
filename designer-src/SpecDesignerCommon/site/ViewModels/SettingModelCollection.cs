using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Collections.Specialized;
using System.Linq;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x0200009A RID: 154
	public class SettingModelCollection : ObservableCollection<SettingModelBase>
	{
		// Token: 0x06000640 RID: 1600 RVA: 0x0001C45D File Offset: 0x0001A65D
		public SettingModelCollection()
		{
		}

		// Token: 0x06000641 RID: 1601 RVA: 0x0001C465 File Offset: 0x0001A665
		public SettingModelCollection(SettingFolderModel parent, IEnumerable<SettingModelBase> items)
		{
			this._parent = parent;
			this.AddRange(items);
		}

		// Token: 0x06000642 RID: 1602 RVA: 0x0001C47B File Offset: 0x0001A67B
		public SettingModelCollection(IEnumerable<SettingModelBase> items)
		{
			this._parent = null;
			this.AddRange(items);
		}

		// Token: 0x06000643 RID: 1603 RVA: 0x0001C491 File Offset: 0x0001A691
		protected override void OnCollectionChanged(NotifyCollectionChangedEventArgs e)
		{
			if (!this.isInAddRange)
			{
				base.OnCollectionChanged(e);
			}
		}

		// Token: 0x06000644 RID: 1604 RVA: 0x0001C4A4 File Offset: 0x0001A6A4
		public void AddRange(IEnumerable<SettingModelBase> collection)
		{
			if (collection == null)
			{
				throw new ArgumentNullException("collection");
			}
			foreach (SettingModelBase settingModelBase in collection)
			{
				settingModelBase.Parent = this._parent;
				base.Items.Add(settingModelBase);
			}
			this.OnCollectionChanged(new NotifyCollectionChangedEventArgs(NotifyCollectionChangedAction.Add, collection.ToList<SettingModelBase>()));
		}

		// Token: 0x06000645 RID: 1605 RVA: 0x0001C520 File Offset: 0x0001A720
		public void RemoveRange(IEnumerable<SettingModelBase> collection)
		{
			if (collection == null)
			{
				throw new ArgumentNullException("collection");
			}
			this.isInAddRange = true;
			foreach (SettingModelBase settingModelBase in collection)
			{
				base.Items.Remove(settingModelBase);
			}
			this.isInAddRange = false;
			this.OnCollectionChanged(new NotifyCollectionChangedEventArgs(NotifyCollectionChangedAction.Remove, collection.ToList<SettingModelBase>()));
		}

		// Token: 0x06000646 RID: 1606 RVA: 0x0001C59C File Offset: 0x0001A79C
		public new void Add(SettingModelBase model)
		{
			model.Parent = this._parent;
			base.Items.Add(model);
		}

		// Token: 0x06000647 RID: 1607 RVA: 0x0001C5B6 File Offset: 0x0001A7B6
		public new void Remove(SettingModelBase model)
		{
			model.Parent = null;
			base.Items.Remove(model);
		}

		// Token: 0x04000268 RID: 616
		private bool isInAddRange;

		// Token: 0x04000269 RID: 617
		private SettingFolderModel _parent;
	}
}
