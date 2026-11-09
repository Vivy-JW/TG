using System;
using System.Collections.ObjectModel;
using System.Collections.Specialized;
using SpecDesigner.Infrastructure.Model;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x02000041 RID: 65
	public class TreeItemCollection : ObservableCollection<TreeItem>
	{
		// Token: 0x0600017B RID: 379 RVA: 0x00007568 File Offset: 0x00005768
		public new void Insert(int index, TreeItem item)
		{
			if (0 > index)
			{
				index = 0;
			}
			if (index > base.Count)
			{
				item.SortIndex = base.Items.Count + 1;
				this.InternalInsert(item);
				return;
			}
			item.SortIndex = index;
			this.InternalInsert(item);
		}

		// Token: 0x0600017C RID: 380 RVA: 0x000075A3 File Offset: 0x000057A3
		protected override void InsertItem(int index, TreeItem item)
		{
			this.Insert(index, item);
		}

		// Token: 0x0600017D RID: 381 RVA: 0x000075AD File Offset: 0x000057AD
		public new void Add(TreeItem item)
		{
			this.InternalInsert(item);
		}

		// Token: 0x0600017E RID: 382 RVA: 0x000075B8 File Offset: 0x000057B8
		private void InternalInsert(TreeItem item)
		{
			if (base.Items.Count == 0)
			{
				base.Items.Add(item);
			}
			else
			{
				bool flag = true;
				for (int i = 0; i < base.Items.Count; i++)
				{
					int num = this._comparison(base.Items[i], item);
					if (num >= 1)
					{
						base.Items.Insert(i, item);
						flag = false;
						break;
					}
				}
				if (flag)
				{
					base.Items.Add(item);
				}
			}
			this.OnCollectionChanged(new NotifyCollectionChangedEventArgs(NotifyCollectionChangedAction.Add, item));
		}

		// Token: 0x040000A0 RID: 160
		private Comparison<TreeItem> _comparison = delegate(TreeItem item, TreeItem target)
		{
			if (item.SortIndex < 0)
			{
				return -1;
			}
			if (item.SortIndex != target.SortIndex)
			{
				return item.SortIndex - target.SortIndex;
			}
			return 1;
		};
	}
}
