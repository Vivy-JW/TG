using System;
using System.Collections.Generic;
using System.Linq;
using System.Windows.Data;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.ViewModel
{
	// Token: 0x02000028 RID: 40
	public class SpecNodesSourceViewModel<T> where T : AbstractSpecNode
	{
		// Token: 0x0600011C RID: 284 RVA: 0x000099C4 File Offset: 0x00007BC4
		public SpecNodesSourceViewModel(IEnumerable<T> specNodes, bool isUncitedOnly)
		{
			this.IsUncitedOnly = isUncitedOnly;
			this._viewSource = new CollectionViewSource();
			this._viewSource.Source = specNodes.ToList<T>();
			this._viewSource.Filter += this.TextFilter;
		}

		// Token: 0x0600011D RID: 285 RVA: 0x00009A11 File Offset: 0x00007C11
		public SpecNodesSourceViewModel(IEnumerable<T> specNodes)
			: this(specNodes, false)
		{
		}

		// Token: 0x17000022 RID: 34
		// (get) Token: 0x0600011E RID: 286 RVA: 0x00009A1B File Offset: 0x00007C1B
		public CollectionViewSource SpecNodesCollection
		{
			get
			{
				return this._viewSource;
			}
		}

		// Token: 0x17000023 RID: 35
		// (get) Token: 0x0600011F RID: 287 RVA: 0x00009A23 File Offset: 0x00007C23
		// (set) Token: 0x06000120 RID: 288 RVA: 0x00009A2B File Offset: 0x00007C2B
		public string FilterText
		{
			get
			{
				return this._filterText;
			}
			set
			{
				this._filterText = value;
				this.SpecNodesCollection.View.Refresh();
			}
		}

		// Token: 0x06000121 RID: 289 RVA: 0x00009A60 File Offset: 0x00007C60
		private void TextFilter(object sender, FilterEventArgs e)
		{
			if (e.Item == null)
			{
				return;
			}
			bool flag = true;
			AbstractSpecNode abstractSpecNode = e.Item as AbstractSpecNode;
			bool isStandardProgram = SettingManager.Get().GetTzpManger(abstractSpecNode.ProgramKey).IsStandardProgram;
			if (this.IsUncitedOnly)
			{
				flag = !isStandardProgram && !abstractSpecNode.IsCited;
			}
			if (flag && !string.IsNullOrEmpty(this._filterText))
			{
				flag = abstractSpecNode.Source.Attributes().Any<XAttribute>((XAttribute a) => a.Value.IndexOf(this._filterText, StringComparison.CurrentCultureIgnoreCase) >= 0) || abstractSpecNode.CDATA.IndexOf(this._filterText, StringComparison.CurrentCultureIgnoreCase) >= 0;
			}
			e.Accepted = flag;
		}

		// Token: 0x04000095 RID: 149
		private bool IsUncitedOnly;

		// Token: 0x04000096 RID: 150
		private CollectionViewSource _viewSource;

		// Token: 0x04000097 RID: 151
		private string _filterText;
	}
}
