using System;
using System.Collections.Generic;
using System.Text.RegularExpressions;
using System.Windows.Data;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.ViewModel
{
	// Token: 0x02000002 RID: 2
	public class FieldsSourceViewModel
	{
		// Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		public FieldsSourceViewModel(PackageKey key)
		{
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(key).SpecificationInfo;
			List<SpecFieldNode> list = null;
			this._viewSource = new CollectionViewSource();
			this._viewSource.Source = list;
			this._viewSource.Filter += this.TextFilter;
		}

		// Token: 0x17000001 RID: 1
		// (get) Token: 0x06000002 RID: 2 RVA: 0x000020A4 File Offset: 0x000002A4
		public CollectionViewSource SpecFieldsCollection
		{
			get
			{
				return this._viewSource;
			}
		}

		// Token: 0x17000002 RID: 2
		// (get) Token: 0x06000003 RID: 3 RVA: 0x000020AC File Offset: 0x000002AC
		// (set) Token: 0x06000004 RID: 4 RVA: 0x000020B4 File Offset: 0x000002B4
		public string FilterText
		{
			get
			{
				return this._filterText;
			}
			set
			{
				this._filterText = value;
				this.SpecFieldsCollection.View.Refresh();
			}
		}

		// Token: 0x06000005 RID: 5 RVA: 0x000020D0 File Offset: 0x000002D0
		private void TextFilter(object sender, FilterEventArgs e)
		{
			if (e.Item == null)
			{
				return;
			}
			SpecFieldNode specFieldNode = e.Item as SpecFieldNode;
			if ((specFieldNode.Status & SpecStatus.DELETE) == SpecStatus.DELETE)
			{
				e.Accepted = false;
				return;
			}
			if (!string.IsNullOrEmpty(this._filterText))
			{
				e.Accepted |= Regex.IsMatch(specFieldNode.CDATA, this._filterText, RegexOptions.IgnoreCase);
				return;
			}
			e.Accepted = true;
		}

		// Token: 0x04000001 RID: 1
		private CollectionViewSource _viewSource;

		// Token: 0x04000002 RID: 2
		private string _filterText;
	}
}
