using System;
using System.Collections.Generic;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.ViewModel
{
	// Token: 0x0200000E RID: 14
	public class UncitedSpecNodesViewModel<T> where T : AbstractSpecNode
	{
		// Token: 0x06000067 RID: 103 RVA: 0x00006380 File Offset: 0x00004580
		public UncitedSpecNodesViewModel(IEnumerable<T> specNodes)
		{
			this._viewSource = new CollectionViewSource();
			this._viewSource.Source = specNodes;
			this._viewSource.Filter += this.TextFilter;
		}

		// Token: 0x1700000A RID: 10
		// (get) Token: 0x06000068 RID: 104 RVA: 0x000063B6 File Offset: 0x000045B6
		public CollectionViewSource SpecNodesCollection
		{
			get
			{
				return this._viewSource;
			}
		}

		// Token: 0x06000069 RID: 105 RVA: 0x000063BE File Offset: 0x000045BE
		private void TextFilter(object sender, FilterEventArgs e)
		{
			if (e.Item == null)
			{
				return;
			}
			e.Accepted = true;
		}

		// Token: 0x04000049 RID: 73
		private CollectionViewSource _viewSource;
	}
}
