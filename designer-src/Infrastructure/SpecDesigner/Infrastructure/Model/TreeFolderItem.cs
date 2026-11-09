using System;
using System.Collections.ObjectModel;
using System.Windows;
using SpecDesignerCommon.Events;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000022 RID: 34
	internal class TreeFolderItem
	{
		// Token: 0x1700001C RID: 28
		// (get) Token: 0x06000071 RID: 113 RVA: 0x00002A8E File Offset: 0x00000C8E
		public ReadOnlyObservableCollection<TreeItem> Nodes
		{
			get
			{
				return new ReadOnlyObservableCollection<TreeItem>(this._nodes);
			}
		}

		// Token: 0x1700001D RID: 29
		// (get) Token: 0x06000072 RID: 114 RVA: 0x00002A9B File Offset: 0x00000C9B
		// (set) Token: 0x06000073 RID: 115 RVA: 0x00002AA3 File Offset: 0x00000CA3
		public string Name { get; set; }

		// Token: 0x06000074 RID: 116 RVA: 0x00002AAC File Offset: 0x00000CAC
		public TreeFolderItem(DefinitionType type)
		{
			switch (type)
			{
			case DefinitionType.FUNCTION:
				this.Name = "FUNCTION";
				return;
			case DefinitionType.DIALOG:
				this.Name = "DIALOG";
				return;
			case DefinitionType.REPORT:
				this.Name = "REPORT";
				return;
			default:
				throw new Exception(Application.Current.FindResource("Message_UnexpectedTreeItem") as string);
			}
		}

		// Token: 0x0400002A RID: 42
		private TreeItemCollection _nodes = new TreeItemCollection();
	}
}
